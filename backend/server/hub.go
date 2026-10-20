package server

import "sync"

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

// getOrCreateTable returns the table with the given name, creating it if needed.
func (h *Hub) getOrCreateTable(name string) *table {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.tables[name]; ok {
		return t
	}
	t := newTable(name)
	go t.run()
	h.tables[name] = t
	return t
}

func (h *Hub) findTableByName(name string) *table {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tables[name]
}
