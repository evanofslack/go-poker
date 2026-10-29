package server

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/evanofslack/go-poker/table"
)

// Store keeps tables so they can be brought back after a restart. A table is
// its state, rewritten at each checkpoint, plus the hands and log lines it
// has added, which are only appended.
type Store interface {
	// Save writes the table's state and appends its new hands and logs, all
	// or nothing.
	Save(ctx context.Context, c Checkpoint) error
	// MarkClosed marks a table closed. It's kept until pruned.
	MarkClosed(ctx context.Context, id, reason string, at time.Time) error
	// Open is every table not closed, with its last table.MaxHands hands and
	// table.MaxLogs log lines.
	Open(ctx context.Context) ([]Saved, error)
	// Prune deletes the tables closed before closedBefore, and says how many.
	// Their audit entries stay.
	Prune(ctx context.Context, closedBefore time.Time) (int, error)

	// AppendAudit adds audit entries, already numbered. An entry written
	// before is left as it was.
	AppendAudit(ctx context.Context, entries []AuditEntry) error
	// Audit is the entries q picks, newest first.
	Audit(ctx context.Context, q AuditQuery) ([]AuditEntry, error)
	// LastAuditID is the highest audit id written, 0 for none.
	LastAuditID(ctx context.Context) (int64, error)
	// PruneAudit deletes the audit entries from before before, and says how
	// many.
	PruneAudit(ctx context.Context, before time.Time) (int, error)
}

// Checkpoint is a table to save: its state, and what it added since it was
// last saved.
type Checkpoint struct {
	ID      string
	Name    string
	Version int
	State   []byte // table.State as JSON
	Hands   []Hand
	Logs    []table.Log
	At      time.Time
}

// Hand is a finished hand: its record and stats.
type Hand struct {
	Record table.HandRecord
	Stats  table.HandStats
}

// Saved is a table as loaded, its hands and logs oldest first.
type Saved struct {
	ID      string
	Name    string
	Version int
	State   []byte
	Hands   []Hand
	Logs    []table.Log
}

// memStore keeps tables in memory, so they don't survive a restart.
type memStore struct {
	mu     sync.Mutex
	tables map[string]*memTable // by id
	audit  []AuditEntry         // oldest first, the last maxMemAudit
}

// maxMemAudit is how many audit entries the memory store keeps.
const maxMemAudit = 10_000

type memTable struct {
	Saved
	closedAt time.Time
	reason   string
}

func newMemStore() *memStore {
	return &memStore{tables: map[string]*memTable{}}
}

func (m *memStore) Save(_ context.Context, c Checkpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.tables[c.ID]
	if t == nil {
		t = &memTable{Saved: Saved{ID: c.ID, Name: c.Name}}
		m.tables[c.ID] = t
	}
	t.Version, t.State = c.Version, c.State
	for _, h := range c.Hands {
		t.Hands = slices.DeleteFunc(t.Hands, func(o Hand) bool { return o.Record.Hand == h.Record.Hand })
		t.Hands = append(t.Hands, h)
	}
	for _, l := range c.Logs {
		t.Logs = slices.DeleteFunc(t.Logs, func(o table.Log) bool { return o.ID == l.ID })
		t.Logs = append(t.Logs, l)
	}
	// nothing outlives the process, so keep only what Open returns
	t.Hands = t.Hands[max(len(t.Hands)-table.MaxHands, 0):]
	t.Logs = t.Logs[max(len(t.Logs)-table.MaxLogs, 0):]
	return nil
}

func (m *memStore) MarkClosed(_ context.Context, id, reason string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t := m.tables[id]; t != nil && t.closedAt.IsZero() {
		t.closedAt, t.reason = at, reason
	}
	return nil
}

func (m *memStore) Open(context.Context) ([]Saved, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Saved
	for _, t := range m.tables {
		if t.closedAt.IsZero() {
			s := t.Saved
			s.Hands, s.Logs = slices.Clone(s.Hands), slices.Clone(s.Logs)
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memStore) Prune(_ context.Context, closedBefore time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, t := range m.tables {
		if !t.closedAt.IsZero() && t.closedAt.Before(closedBefore) {
			delete(m.tables, id)
			n++
		}
	}
	return n, nil
}

func (m *memStore) AppendAudit(_ context.Context, entries []AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range entries {
		if n := len(m.audit); n == 0 || e.ID > m.audit[n-1].ID {
			m.audit = append(m.audit, e)
		}
	}
	m.audit = m.audit[max(len(m.audit)-maxMemAudit, 0):]
	return nil
}

func (m *memStore) Audit(_ context.Context, q AuditQuery) ([]AuditEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []AuditEntry
	for i := len(m.audit) - 1; i >= 0 && len(out) < q.limit(); i-- {
		if q.match(m.audit[i]) {
			out = append(out, m.audit[i])
		}
	}
	return out, nil
}

func (m *memStore) LastAuditID(context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n := len(m.audit); n > 0 {
		return m.audit[n-1].ID, nil
	}
	return 0, nil
}

func (m *memStore) PruneAudit(_ context.Context, before time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.audit)
	m.audit = slices.DeleteFunc(m.audit, func(e AuditEntry) bool { return e.Time < before.UnixMilli() })
	return n - len(m.audit), nil
}
