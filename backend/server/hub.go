package server

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/evanofslack/go-poker/table"
)

// Hub maintains the set of active clients and tables.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	stop       chan chan []*Client // shutdown asks run for the clients to close
	stopping   bool                // owned by run, like clients

	mu     sync.Mutex // guards tables
	tables map[string]*tableActor

	connMu sync.Mutex     // guards conns and total, counted before the upgrade
	conns  map[string]int // open connections by address
	total  int

	store  Store
	timing table.Timing
}

func newHub(store Store) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan chan []*Client),
		tables:     make(map[string]*tableActor),
		conns:      make(map[string]int),
		store:      store,
		timing:     timing,
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case reply := <-h.stop:
			h.stopping = true
			var clients []*Client
			for c := range h.clients {
				close(c.quit)
				clients = append(clients, c)
			}
			reply <- clients
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.clients[client] = true
	if h.stopping {
		close(client.quit)
	}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

// maxTables caps how many tables can exist at once.
var maxTables = 100

var (
	errTooManyTables   = errors.New("too many tables")
	errTooManyTablesIP = errors.New("you have too many open tables")
	errTableExists     = errors.New("table already exists")
	errNoTable         = errors.New("table not found")
)

// restore brings back the tables in the store, as after a restart.
func (h *Hub) restore() error {
	saved, err := h.store.Load()
	if err != nil {
		return err
	}
	now := time.Now()
	for name, b := range saved {
		var s table.State
		if err := json.Unmarshal(b, &s); err != nil || s.Version != table.Version {
			slog.Default().Warn("dropping saved table", "table", name, "error", err)
			h.store.Delete(name)
			continue
		}
		table.Restore(&s, now)
		a := newTableActor(h, &s)
		h.tables[name] = a
		go a.run()
	}
	return nil
}

// createTable makes a table named name for the client at ip.
func (h *Hub) createTable(name, ip, password string, settings table.Settings) (*tableActor, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch {
	case h.tables[name] != nil:
		return nil, errTableExists
	case len(h.tables) >= maxTables:
		limitHits.WithLabelValues("tables").Inc()
		return nil, errTooManyTables
	case h.tablesFrom(ip) >= maxTablesPerIP:
		limitHits.WithLabelValues("tables_per_ip").Inc()
		return nil, errTooManyTablesIP
	}
	a := newTableActor(h, table.New(name, settings, h.timing, password, ip, time.Now()))
	h.tables[name] = a
	go a.run()
	return a, nil
}

func (h *Hub) table(name string) *tableActor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tables[name]
}

// removeTable forgets a, which has closed.
func (h *Hub) removeTable(a *tableActor, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tables[a.name] == a {
		delete(h.tables, a.name)
		tablesClosed.WithLabelValues(reason).Inc()
	}
}

// tablesFrom counts the open tables created from ip. h.mu must be held.
func (h *Hub) tablesFrom(ip string) int {
	n := 0
	for _, a := range h.tables {
		if a.creatorIP == ip {
			n++
		}
	}
	return n
}

func (h *Hub) allTables() []*tableActor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Collect(maps.Values(h.tables))
}

// reserveConn counts a connection from ip, unless that would go over a
// limit, in which case it reports the limit.
func (h *Hub) reserveConn(ip string) (ok bool, limit string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	if h.total >= maxConns {
		return false, "conns"
	}
	if h.conns[ip] >= maxConnsPerIP {
		return false, "conns_per_ip"
	}
	h.total++
	h.conns[ip]++
	return true, ""
}

// releaseConn frees a connection counted by reserveConn.
func (h *Hub) releaseConn(ip string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.total--
	if h.conns[ip]--; h.conns[ip] <= 0 {
		delete(h.conns, ip)
	}
}

// shutdown closes every table, telling its players the server is restarting,
// then closes every connection once its pending messages are written, or ctx
// is done.
func (h *Hub) shutdown(ctx context.Context) {
	for _, a := range h.allTables() {
		a.shutdown()
	}

	reply := make(chan []*Client, 1)
	h.stop <- reply
	for _, c := range <-reply {
		select {
		case <-c.done:
		case <-ctx.Done():
			return
		}
	}
}

// publicTables summarizes the public tables whose session hasn't ended, with
// the most players first and full tables last.
func (h *Hub) publicTables() []table.Summary {
	tables := []table.Summary{}
	for _, a := range h.allTables() {
		if a.public && !a.ended.Load() {
			tables = append(tables, *a.summary.Load())
		}
	}
	slices.SortFunc(tables, func(a, b table.Summary) int {
		return cmp.Or(
			cmp.Compare(boolInt(a.Players >= a.Seats), boolInt(b.Players >= b.Seats)),
			cmp.Compare(b.Players, a.Players),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return tables
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
