package server

import "github.com/evanofslack/go-poker/poker"

// table is a single table or game of poker
type table struct {
	name       string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	game       *poker.Game
}

// newTable creates a new table
func newTable(name string) *table {
	return &table{
		name:       name,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		game:       poker.NewGame(),
	}
}

func (t *table) run() {
	for {
		select {
		case client := <-t.register:
			t.registerClient(client)
		case client := <-t.unregister:
			t.unregisterClient(client)
		case message := <-t.broadcast:
			t.broadcastToClients(message)
		}
	}
}

func (t *table) registerClient(client *Client) {
	t.clients[client] = true
}

func (t *table) unregisterClient(client *Client) {
	if _, ok := t.clients[client]; ok {
		delete(t.clients, client)
		close(client.send)
	}
}

func (t *table) broadcastToClients(message []byte) {
	for client := range t.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(t.clients, client)
		}
	}
}
