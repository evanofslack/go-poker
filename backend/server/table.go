package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/evanofslack/go-poker/holdem"
	"github.com/evanofslack/go-poker/table"
)

var errClosed = errors.New("table not found")

// tableActor runs one table. Its goroutine owns the state and the clients
// bound to it; everyone else talks to it through the inbox.
type tableActor struct {
	name      string
	hub       *Hub
	creatorIP string // fixed at creation
	inbox     chan envelope
	done      chan struct{} // closed once the table has closed

	// read by the hub without going through the inbox
	summary atomic.Pointer[table.Summary]
	ended   atomic.Bool
	public  atomic.Bool

	// owned by run
	state  *table.State
	conns  map[string]*Client // by person id
	slow   map[*Client]bool   // dropped for not keeping up, waiting to disconnect
	armed  map[table.Timer]armedTimer
	closed bool

	// the last hand and log line in the store
	savedHand int
	savedLog  int
}

type armedTimer struct {
	at    time.Time
	timer *time.Timer
}

// envelope is a command for the table, or a shutdown when cmd is nil. client
// is who sent it, for a Join, Leave or Disconnect; reply, when set, gets the
// result once the command is done.
type envelope struct {
	cmd    table.Command
	client *Client
	reply  chan reply
}

type reply struct {
	joined *table.Joined
	err    error
}

func newTableActor(h *Hub, s *table.State) *tableActor {
	a := &tableActor{
		name:      s.Name,
		hub:       h,
		creatorIP: s.CreatorIP,
		inbox:     make(chan envelope, 64),
		done:      make(chan struct{}),
		state:     s,
		conns:     map[string]*Client{},
		slow:      map[*Client]bool{},
		armed:     map[table.Timer]armedTimer{},
	}
	if n := len(s.History); n > 0 {
		a.savedHand = s.History[n-1].Hand
	}
	if n := len(s.Logs); n > 0 {
		a.savedLog = s.Logs[n-1].ID
	}
	a.publish()
	return a
}

// ask sends cmd and waits for it to be done.
func (a *tableActor) ask(cmd table.Command, c *Client) reply {
	r := make(chan reply, 1)
	select {
	case a.inbox <- envelope{cmd, c, r}:
	case <-a.done:
		return reply{err: errClosed}
	}
	select {
	case res := <-r:
		return res
	case <-a.done:
		return reply{err: errClosed}
	}
}

// post sends cmd without waiting, dropping it if the table closed.
func (a *tableActor) post(cmd table.Command) {
	select {
	case a.inbox <- envelope{cmd: cmd}:
	case <-a.done:
	}
}

// shutdown saves the table, even mid-hand, and closes it for a server
// restart.
func (a *tableActor) shutdown() {
	a.ask(nil, nil)
}

func (a *tableActor) run() {
	a.arm()
	for {
		select {
		case env := <-a.inbox:
			a.handle(env)
			if a.closed {
				return
			}
		case <-a.done:
			return
		}
	}
}

func (a *tableActor) handle(env envelope) {
	var r reply
	defer func() {
		if env.reply != nil {
			env.reply <- r
		}
	}()
	if env.cmd == nil {
		a.save()
		a.close("shutdown", table.Error{Code: table.CodeRestarting, Message: "the server is restarting"}, false)
		return
	}
	if d, ok := env.cmd.(table.Disconnect); ok && a.conns[d.Person] != env.client {
		return // a connection already replaced by a reconnect
	}

	now := time.Now()
	res, err := a.reduce(now, env.cmd)
	if err != nil {
		r.err = err
		return
	}
	// the lobby sees the change before anyone hears of it, so a client that
	// lists tables on an update finds it there
	a.publish()
	if res.Joined != nil {
		r.joined = res.Joined
		a.bind(res.Joined, env.client)
		a.welcome(res.Joined.ID, env.cmd.(table.Join).Password, env.client, now)
	}
	for _, o := range res.Out {
		a.deliver(o, env)
	}
	a.prune()
	if res.Changed {
		a.update(res.Events, now)
	}
	a.metrics(res)
	a.arm()
	if res.Checkpoint {
		a.save()
	}
	if res.Close != nil {
		a.close(res.Close.Reason, table.Error{Code: table.CodeClosed, Message: res.Close.Message}, true)
	}
}

// reduce applies cmd, recovering from a panic so one bad command can't take
// the server down.
func (a *tableActor) reduce(now time.Time, cmd table.Command) (res table.Result, err error) {
	defer func() {
		if p := recover(); p != nil {
			panicsTotal.Inc()
			slog.Default().Error("table panic", "table", a.name, "command", fmt.Sprintf("%T", cmd), "panic", p)
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return table.Reduce(a.state, now, cmd), nil
}

// bind attaches c to the person it joined as. Reclaiming a seat drops the
// person's old connection, like a stale tab.
func (a *tableActor) bind(j *table.Joined, c *Client) {
	if old := a.conns[j.ID]; old != nil && old != c {
		old.conn.Close()
		delete(a.slow, old)
	}
	a.conns[j.ID] = c
}

// prune detaches the connections of people no longer at the table: who left,
// were removed, or disconnected.
func (a *tableActor) prune() {
	for id, c := range a.conns {
		if p := a.state.People[id]; p == nil || !p.Connected {
			delete(a.conns, id)
			delete(a.slow, c)
		}
	}
}

// deliver sends o to who it's for. A message to someone not yet joined goes
// to the client that tried.
func (a *tableActor) deliver(o table.Out, env envelope) {
	msg := marshalOut(o.Msg)
	if msg == nil {
		return
	}
	switch {
	case o.To == "":
		for _, c := range a.conns {
			a.sendTo(c, msg)
		}
	case a.conns[o.To] != nil:
		a.sendTo(a.conns[o.To], msg)
	case env.client != nil:
		// not joined yet, or just left or removed
		a.sendTo(env.client, msg)
	}
}

func marshalOut(msg any) []byte {
	switch m := msg.(type) {
	case table.Error:
		return marshal(ErrorOut{ActionError, m.Code, m.Message})
	case table.ChatMessage:
		return marshal(ChatOut{ActionChat, m})
	case table.NewToken:
		return marshal(TokenOut{ActionToken, m.Token})
	case table.NewPassword:
		return marshal(PasswordOut{ActionPassword, m.Password})
	case table.HandsPage:
		return marshal(HandHistory{ActionHandHistory, m.Hands, m.More})
	}
	slog.Default().Warn("unknown table message", "type", fmt.Sprintf("%T", msg))
	return nil
}

func marshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Default().Warn("marshal", "error", err)
		return nil
	}
	return b
}

// welcome sends someone who just joined the table so far. password is what
// they joined with, for sharing invite links.
func (a *tableActor) welcome(id, password string, c *Client, now time.Time) {
	s := a.state
	p := s.People[id]
	history, more := table.HistoryFor(s, p.Name, 0)
	a.sendTo(c, marshal(Joined{ActionJoined, s.Name, p.Name, id, password}))
	a.sendTo(c, marshal(Welcome{
		Action:  ActionWelcome,
		View:    table.ViewFor(s, id, now),
		Logs:    nonNil(s.Logs),
		Stats:   nonNil(s.Stats),
		Totals:  table.TotalsView(s),
		History: history,
		More:    more,
	}))
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// update sends everyone their view of the table and what happened.
func (a *tableActor) update(events []table.Event, now time.Time) {
	for id, c := range a.conns {
		u := TableUpdate{Action: ActionTableUpdate, View: table.ViewFor(a.state, id, now), Events: []EventEnvel{}}
		for _, e := range table.EventsFor(a.state, events, id) {
			u.Events = append(u.Events, EventEnvel{e.EventType(), marshal(e)})
		}
		a.sendTo(c, marshal(u))
	}
}

// sendTo queues msg for c, dropping a client too slow to keep up rather than
// stalling the table. Its read pump then disconnects it.
func (a *tableActor) sendTo(c *Client, msg []byte) {
	if a.slow[c] {
		return
	}
	select {
	case c.send <- msg:
	default:
		slowClientDrops.Inc()
		a.slow[c] = true
		c.conn.Close()
	}
}

func (a *tableActor) metrics(res table.Result) {
	for _, e := range res.Events {
		if _, ok := e.(holdem.HandStarted); ok {
			handsTotal.Inc()
		}
	}
	if res.Grace != "" {
		seatGrace.WithLabelValues(res.Grace).Inc()
	}
}

// arm sets a timer for each deadline in the state, replacing ones that moved
// and stopping ones that went away.
func (a *tableActor) arm() {
	for t, armed := range a.armed {
		if at, ok := a.state.Timers[t]; !ok || !at.Equal(armed.at) {
			armed.timer.Stop()
			delete(a.armed, t)
		}
	}
	for t, at := range a.state.Timers {
		if _, ok := a.armed[t]; ok {
			continue
		}
		t, at := t, at
		a.armed[t] = armedTimer{at, time.AfterFunc(time.Until(at), func() {
			a.post(table.TimerFired{Timer: t, At: at})
		})}
	}
}

// saveTimeout bounds a save, which holds up the table.
const saveTimeout = 5 * time.Second

// save writes the table to the store with the hands and log lines the store
// doesn't have yet. A failed save is retried by the next one.
func (a *tableActor) save() {
	s := a.state
	b, err := json.Marshal(s)
	if err != nil {
		slog.Default().Error("marshal table", "table", a.name, "error", err)
		return
	}
	c := Checkpoint{ID: s.ID, Name: s.Name, Version: s.Version, State: b, At: time.Now()}
	// the last hand saved goes again: showing cards after it, and what that
	// pays, change its record
	for i, h := range s.History {
		if h.Hand >= a.savedHand {
			c.Hands = append(c.Hands, Hand{h, s.Stats[i]})
		}
	}
	for _, l := range s.Logs {
		if l.ID > a.savedLog {
			c.Logs = append(c.Logs, l)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
	defer cancel()
	if err := a.hub.store.Save(ctx, c); err != nil {
		saveErrors.Inc()
		slog.Default().Error("save table", "table", a.name, "error", err)
		return
	}
	saveSeconds.Observe(time.Since(c.At).Seconds())
	if n := len(c.Hands); n > 0 {
		a.savedHand = c.Hands[n-1].Record.Hand
	}
	if n := len(c.Logs); n > 0 {
		a.savedLog = c.Logs[n-1].ID
	}
}

// publish updates what the hub reads without asking the table.
func (a *tableActor) publish() {
	sum := table.Summarize(a.state)
	a.summary.Store(&sum)
	a.ended.Store(a.state.Ended())
	a.public.Store(a.state.Settings.Public)
}

// close tells everyone why the table is going, stops it and removes it from
// the hub. forget marks it closed in the store; a shutdown leaves it open.
func (a *tableActor) close(reason string, why table.Error, forget bool) {
	msg := marshalOut(why)
	for _, c := range a.conns {
		a.sendTo(c, msg)
	}
	for _, armed := range a.armed {
		armed.timer.Stop()
	}
	a.closed = true
	a.hub.removeTable(a, reason)
	if forget {
		ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
		defer cancel()
		if err := a.hub.store.MarkClosed(ctx, a.state.ID, reason, time.Now()); err != nil {
			saveErrors.Inc()
			slog.Default().Error("close table", "table", a.name, "error", err)
		}
	}
	close(a.done)
}
