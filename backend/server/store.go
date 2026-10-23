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
	Prune(ctx context.Context, closedBefore time.Time) (int, error)
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
}

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
