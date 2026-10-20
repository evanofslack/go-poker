package server

import (
	"sync"

	"github.com/evanofslack/go-poker/poker"
)

// stateUpdate is a game update with one view per player, since each player
// may only see their own cards. Clients without a seat get the spectator view.
type stateUpdate struct {
	byUUID    map[string][]byte
	spectator []byte
}

// table is a single table or game of poker
type table struct {
	name       string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	updates    chan stateUpdate
	game       *poker.Game
	done       chan struct{} // closed when the table is deleted
	members    int           // clients at this table, guarded by Hub.mu

	mu sync.Mutex // serializes game actions from client goroutines
}

// newTable creates a new table
func newTable(name string) *table {
	return &table{
		name:       name,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		updates:    make(chan stateUpdate),
		game:       poker.NewGame(),
		done:       make(chan struct{}),
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
		case u := <-t.updates:
			t.sendUpdate(u)
		case <-t.done:
			return
		}
	}
}

func (t *table) registerClient(client *Client) {
	t.clients[client] = true
}

func (t *table) unregisterClient(client *Client) {
	if _, ok := t.clients[client]; ok {
		delete(t.clients, client)
	}
}

func (t *table) broadcastToClients(message []byte) {
	for client := range t.clients {
		t.sendTo(client, message)
	}
}

func (t *table) sendUpdate(u stateUpdate) {
	for client := range t.clients {
		msg, ok := u.byUUID[client.id()]
		if !ok {
			msg = u.spectator
		}
		t.sendTo(client, msg)
	}
}

func (t *table) sendTo(client *Client, message []byte) {
	select {
	case client.send <- message:
	default:
		// slow client: drop it and let its readPump unregister it,
		// the hub is the only one that closes client.send
		delete(t.clients, client)
		client.conn.Close()
	}
}
