package server

import "github.com/go-redis/redis/v8"

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	rdb        *redis.Client
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	tables     map[*table]bool
}

func newHub() *Hub {
	return &Hub{
		rdb:        newRedisClient(),
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		tables:     make(map[*table]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastToClients(message)
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

func (h *Hub) broadcastToClients(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) createTable(name string) *table {
	table := newTable(name, h.rdb)
	go table.run()
	h.tables[table] = true
	return table
}

func (h *Hub) findTableByName(name string) *table {
	var foundTable *table
	for table := range h.tables {
		if table.name == name {
			foundTable = table
		}
	}
	return foundTable
}
