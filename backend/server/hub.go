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

var errTooManyTables = errors.New("too many tables")

// joinTable moves c to the named table, creating the table if needed.
func (h *Hub) joinTable(c *Client, name string) error {
	if c.table != nil && c.table.name == name {
		return nil
	}

	h.mu.Lock()
	t, ok := h.tables[name]
	if !ok {
		if len(h.tables) >= maxTables {
			h.mu.Unlock()
			return errTooManyTables
		}
		t = newTable(name)
		go t.run()
		h.tables[name] = t
	}
	t.members++
	h.mu.Unlock()

	h.leaveTable(c)
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
