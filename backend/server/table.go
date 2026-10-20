package server

import (
	"strings"
	"sync"
	"time"

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

	mu    sync.Mutex       // serializes game actions from client goroutines
	users map[*Client]bool // clients at this table, guarded by mu
	host  *Client          // guarded by mu

	dealTimer *time.Timer     // deals the next hand, guarded by mu
	turnTimer *time.Timer     // acts for an idle player, guarded by mu
	turnSeq   int             // invalidates old turn timers, guarded by mu
	leaving   map[string]bool // uuids of players who left, guarded by mu
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
		users:      make(map[*Client]bool),
		leaving:    make(map[string]bool),
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

// nameTaken reports whether name is used by anyone at the table, ignoring
// case. Caller holds t.mu.
func (t *table) nameTaken(name string) bool {
	for u := range t.users {
		if strings.EqualFold(u.username, name) {
			return true
		}
	}
	// seated players may no longer be connected
	for _, p := range t.game.GenerateOmniView().Players {
		if strings.EqualFold(p.Username, name) {
			return true
		}
	}
	return false
}

// removeUser takes c out of the table's users, passing the host role on if c
// had it: to a seated user first, else anyone. Caller holds t.mu.
func (t *table) removeUser(c *Client) {
	delete(t.users, c)
	if t.host != c {
		return
	}
	t.host = nil
	for _, p := range t.game.GenerateOmniView().Players {
		for u := range t.users {
			if u.id() == p.UUID {
				t.host = u
				return
			}
		}
	}
	for u := range t.users {
		t.host = u
		return
	}
}

// hostID is the host's uuid, or empty if there is none. Caller holds t.mu.
func (t *table) hostID() string {
	if t.host == nil {
		return ""
	}
	return t.host.id()
}

// send queues message for every client, unless the table has been deleted.
func (t *table) send(message []byte) {
	select {
	case t.broadcast <- message:
	case <-t.done:
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
