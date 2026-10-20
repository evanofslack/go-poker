package server

import (
	"errors"
	"sync"
)

// Hub maintains the set of active clients and tables.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client

	mu     sync.Mutex // guards tables, which client goroutines access directly
	tables map[string]*table
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		tables:     make(map[string]*table),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.clients[client] = true
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

// maxTables caps how many tables can exist at once.
const maxTables = 100

var (
	errTooManyTables = errors.New("too many tables")
	errTableExists   = errors.New("table already exists")
	errNoTable       = errors.New("table not found")
	errNameTaken     = errors.New("name already taken at this table")
	errAtTable       = errors.New("already at a table")
)

// joinTable adds c to the named table as username. With create, the table
// must not exist yet and c becomes its host; otherwise it must exist.
func (h *Hub) joinTable(c *Client, name string, username string, create bool) error {
	if c.table != nil {
		return errAtTable
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	t, ok := h.tables[name]
	switch {
	case create && ok:
		return errTableExists
	case !create && !ok:
		return errNoTable
	case create && len(h.tables) >= maxTables:
		return errTooManyTables
	}

	// lock order is hub.mu, then table.mu
	if create {
		t = newTable(name)
	}
	t.mu.Lock()
	if t.nameTaken(username) {
		t.mu.Unlock()
		return errNameTaken
	}
	c.username = username
	t.users[c] = true
	if create {
		t.host = c
	}
	t.mu.Unlock()

	if create {
		go t.run()
		h.tables[name] = t
	}
	t.members++
	c.table = t
	t.register <- c
	return nil
}

// leaveTable removes c from its table, and deletes the table once empty.
func (h *Hub) leaveTable(c *Client) {
	t := c.table
	if t == nil {
		return
	}
	c.table = nil
	t.unregister <- c

	t.mu.Lock()
	t.removeUser(c)
	if t.host != nil {
		broadcastGame(t) // the host may have changed
	}
	t.mu.Unlock()

	h.mu.Lock()
	t.members--
	empty := t.members == 0
	if empty {
		delete(h.tables, t.name)
	}
	h.mu.Unlock()

	if empty {
		close(t.done)
	}
}
