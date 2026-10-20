package server

import (
	"errors"
	"fmt"
	"sync"
	"time"
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
// must not exist yet and c becomes its host; otherwise it must exist. A valid
// token reclaims a disconnected player's seat, and their name.
func (h *Hub) joinTable(c *Client, name string, username string, token string, create bool) error {
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
	if uuid, seatName, ok := t.reclaim(token); ok && !create {
		// another connection may still hold the seat, e.g. a stale tab
		for u := range t.users {
			if u.id() == uuid {
				u.uuid.Store(newUUID())
				u.conn.Close()
			}
		}
		if timer, ok := t.grace[uuid]; ok {
			timer.Stop()
			delete(t.grace, uuid)
		}
		c.uuid.Store(uuid)
		username = seatName
	} else if t.nameTaken(username) {
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

// leaveTable removes c from its table, and deletes the table once empty. A
// seated player who disconnected keeps their seat for reconnectGrace.
func (h *Hub) leaveTable(c *Client, disconnected bool) {
	t := c.table
	if t == nil {
		return
	}
	c.table = nil
	t.unregister <- c

	t.mu.Lock()
	uuid := c.id()
	if disconnected && isSeatedUUID(t, uuid) {
		t.send(createNewLog(fmt.Sprintf("%s disconnected", c.username)))
		t.grace[uuid] = time.AfterFunc(reconnectGrace, func() { h.graceExpired(t, uuid) })
	} else {
		leaveSeat(t, uuid)
	}
	t.removeUser(c)
	if t.host != nil {
		broadcastGame(t) // the host may have changed
	}
	t.mu.Unlock()

	h.mu.Lock()
	t.members--
	h.mu.Unlock()
	h.deleteIfEmpty(t)
}

// graceExpired removes a disconnected player who didn't come back in time.
func (h *Hub) graceExpired(t *table, uuid string) {
	t.mu.Lock()
	if _, ok := t.grace[uuid]; ok {
		delete(t.grace, uuid)
		leaveSeat(t, uuid)
	}
	t.mu.Unlock()
	h.deleteIfEmpty(t)
}

// deleteIfEmpty deletes t once nobody is connected and no seat is waiting
// for a reconnect.
func (h *Hub) deleteIfEmpty(t *table) {
	h.mu.Lock()
	t.mu.Lock()
	empty := t.members == 0 && len(t.grace) == 0 && h.tables[t.name] == t
	t.mu.Unlock()
	if empty {
		delete(h.tables, t.name)
	}
	h.mu.Unlock()

	if empty {
		close(t.done)
	}
}
