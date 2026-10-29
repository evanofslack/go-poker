package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/evanofslack/go-poker/table"
)

// AuditEntry is something a connection did or tried, for the operator: who,
// from where, and whether it was refused. Players never see these.
type AuditEntry struct {
	ID        int64           `json:"id"`        // assigned when written, in order
	Time      int64           `json:"time"`      // unix ms
	Table     string          `json:"table"`     // table id, empty before joining one
	TableName string          `json:"tableName"` // empty when the message named none
	Person    string          `json:"person"`    // person id, empty before joining
	Name      string          `json:"name"`
	IP        string          `json:"ip"`
	Conn      string          `json:"conn"`  // the connection's id
	Kind      string          `json:"kind"`  // the command, like "take-seat", or "limit"
	Error     string          `json:"error"` // why it was refused, empty if it wasn't
	Data      json.RawMessage `json:"data,omitempty"`
}

// AuditQuery picks audit entries, newest first. Empty fields match anything.
type AuditQuery struct {
	Table   string // a table id or name
	Person  string
	Name    string
	IP      string
	Kind    string
	Refused bool  // only refused ones
	Before  int64 // ids below this, 0 for the newest
	Limit   int   // at most maxAuditPage, which is also the default
}

// maxAuditPage is the most entries one query returns.
const maxAuditPage = 200

func (q AuditQuery) limit() int {
	if q.Limit <= 0 || q.Limit > maxAuditPage {
		return maxAuditPage
	}
	return q.Limit
}

func (q AuditQuery) match(e AuditEntry) bool {
	return (q.Table == "" || e.Table == q.Table || e.TableName == q.Table) &&
		(q.Person == "" || e.Person == q.Person) &&
		(q.Name == "" || e.Name == q.Name) &&
		(q.IP == "" || e.IP == q.IP) &&
		(q.Kind == "" || e.Kind == q.Kind) &&
		(!q.Refused || e.Error != "") &&
		(q.Before == 0 || e.ID < q.Before)
}

const (
	auditEvery    = time.Second // how often entries are written
	auditBatch    = 100         // or as soon as this many are waiting
	auditBacklog  = 10_000      // entries kept while the store is failing
	auditQueueLen = 1024
)

// audit queues e to be written, dropping it rather than wait.
func (h *Hub) audit(e AuditEntry) {
	if e.Time == 0 {
		e.Time = time.Now().UnixMilli()
	}
	select {
	case h.audits <- e:
	default:
		auditDropped.Inc()
	}
}

// writeAudits numbers the queued entries and writes them in batches. A
// failed batch is tried again with the next.
func (h *Hub) writeAudits() {
	ctx := context.Background()
	var last int64
	numbered := false
	var batch []AuditEntry
	write := func() {
		if len(batch) == 0 {
			return
		}
		if !numbered {
			var err error
			if last, err = h.store.LastAuditID(ctx); err != nil {
				slog.Default().Error("read last audit id", "error", err)
				return
			}
			numbered = true
		}
		for i := range batch {
			if batch[i].ID == 0 {
				last++
				batch[i].ID = last
			}
		}
		wctx, cancel := context.WithTimeout(ctx, saveTimeout)
		defer cancel()
		if err := h.store.AppendAudit(wctx, batch); err != nil {
			saveErrors.Inc()
			slog.Default().Error("write audit", "count", len(batch), "error", err)
			if n := len(batch) - auditBacklog; n > 0 {
				auditDropped.Add(float64(n))
				batch = batch[n:]
			}
			return
		}
		batch = batch[:0]
	}

	tick := time.NewTicker(auditEvery)
	defer tick.Stop()
	for {
		select {
		case e := <-h.audits:
			batch = append(batch, e)
			if len(batch) >= auditBatch {
				write()
			}
		case <-tick.C:
			write()
		case done := <-h.flushAudits:
			for len(h.audits) > 0 {
				batch = append(batch, <-h.audits)
			}
			write()
			close(done)
		}
	}
}

// flushAudit writes what's queued, waiting until it's done or ctx is.
func (h *Hub) flushAudit(ctx context.Context) {
	done := make(chan struct{})
	select {
	case h.flushAudits <- done:
	case <-ctx.Done():
		return
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// auditCommand records what cmd from c did at the table: refused with err,
// or done, with the log lines it added. Bets and hand history pages aren't
// recorded unless refused: the hands already have every bet.
func (a *tableActor) auditCommand(c *Client, cmd table.Command, person, name string, logSeq int, res table.Result) {
	kind := commandKind(cmd)
	if res.Err == nil && (strings.HasPrefix(kind, "play-") || kind == "get-hands") {
		return
	}
	e := AuditEntry{Table: a.id, TableName: a.name, Person: person, Name: name, IP: c.ip, Conn: c.id, Kind: kind}
	data := map[string]any{"command": redact(cmd)}
	if res.Err != nil {
		e.Error = res.Err.Message
	}
	if j := res.Joined; j != nil {
		e.Person, e.Name = j.ID, j.Name
		data["reclaimed"] = j.Reclaimed
	}
	if j, ok := cmd.(table.Join); ok {
		if e.Name == "" {
			e.Name = j.Name
		}
		data["token"] = j.Token != ""
	}
	var logs []string
	for _, l := range a.state.Logs {
		if l.ID > logSeq {
			logs = append(logs, l.Message)
		}
	}
	if logs != nil {
		data["logs"] = logs
	}
	e.Data, _ = json.Marshal(data)
	a.hub.audit(e)
}

// auditRefused records a message from c refused before reaching a table.
func (c *Client) auditRefused(in Inbound, err error) {
	e := AuditEntry{Person: c.person, IP: c.ip, Conn: c.id, Kind: in.Action, Error: err.Error()}
	if e.Kind == "" {
		e.Kind = "bad-message"
	} else {
		in.Token, in.Password = "", ""
		e.Data, _ = json.Marshal(in)
	}
	if c.table != nil {
		e.Table, e.TableName = c.table.id, c.table.name
	} else {
		e.TableName, e.Name = in.Tablename, in.Username
	}
	c.hub.audit(e)
}

// redact is cmd without the passwords and tokens in it.
func redact(cmd table.Command) table.Command {
	switch c := cmd.(type) {
	case table.Join:
		c.Password, c.Token = "", ""
		return c
	case table.SetPassword:
		c.Password = ""
		return c
	}
	return cmd
}

// commandKind names cmd in kebab-case, a bet by its action: "take-seat",
// "play-raise".
func commandKind(cmd table.Command) string {
	if p, ok := cmd.(table.Play); ok && p.Action != nil {
		return "play-" + kebab(reflect.TypeOf(p.Action).Name())
	}
	return kebab(reflect.TypeOf(cmd).Name())
}

func kebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
