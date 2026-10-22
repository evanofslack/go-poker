package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"
)

// Hub maintains the set of active clients and tables.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	stop       chan chan []*Client // shutdown asks run for the clients to close
	stopping   bool                // owned by run, like clients

	mu     sync.Mutex // guards tables, which client goroutines access directly
	tables map[string]*table

	connMu sync.Mutex     // guards conns and total, counted before the upgrade
	conns  map[string]int // open connections by address
	total  int
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan chan []*Client),
		tables:     make(map[string]*table),
		conns:      make(map[string]int),
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
	errTooManyTables = errors.New("too many tables")
	errTableExists   = errors.New("table already exists")
	errNoTable       = errors.New("table not found")
	errNameTaken     = errors.New("name already taken at this table")
	errAtTable       = errors.New("already at a table")
	errNeedPassword  = errors.New("this table needs a password")
	errWrongPassword = errors.New("wrong password")
)

// errTooManyTablesIP caps tables per creator, so one person can't take them all.
var errTooManyTablesIP = errors.New("you have too many open tables")

// errorCodes lets the client tell apart the errors it handles
var errorCodes = map[error]string{
	errNeedPassword:  "password-required",
	errWrongPassword: "wrong-password",
}

// codes that tell a client to leave the table: the host removed it, or the
// table closed an hour after its session ended
const (
	codeRemoved = "removed"
	codeClosed  = "closed"
)

// joinTable adds c to the named table as username. With create, the table
// must not exist yet, is made with settings and password, and c becomes its
// host; otherwise it must exist and password must match. A valid token
// reclaims a disconnected player's seat, and their name, without the
// password, and reports so.
func (h *Hub) joinTable(c *Client, name, username, token, password string, create bool, settings tableSettings) (bool, error) {
	if c.table != nil {
		return false, errAtTable
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	t, ok := h.tables[name]
	switch {
	case create && ok:
		return false, errTableExists
	case !create && !ok:
		return false, errNoTable
	case create && len(h.tables) >= maxTables:
		return false, errTooManyTables
	case create && h.tablesFrom(c.ip) >= maxTablesPerIP:
		return false, errTooManyTablesIP
	}

	// lock order is hub.mu, then table.mu
	if create {
		t = newTable(name, settings)
		t.password = password
		t.hub = h
		t.creatorIP = c.ip
	}
	t.mu.Lock()
	uuid, seatName, reconnected := t.reclaim(token)
	reconnected = reconnected && !create
	if reconnected {
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
		t.log(fmt.Sprintf("%s reconnected", seatName))
	} else if err := t.checkPassword(password); err != nil {
		t.mu.Unlock()
		return false, err
	} else if t.nameTaken(username) {
		t.mu.Unlock()
		return false, errNameTaken
	}
	c.username = username
	t.users[c] = true
	t.lastActive = time.Now()
	if create || t.host == "" {
		t.host = username
	}
	t.mu.Unlock()

	if create {
		go t.run()
		h.tables[name] = t
	}
	t.members++
	c.table = t
	t.register <- c
	return reconnected, nil
}

// tablesFrom counts the open tables created from ip. h.mu must be held.
func (h *Hub) tablesFrom(ip string) int {
	n := 0
	for _, t := range h.tables {
		if t.creatorIP == ip {
			n++
		}
	}
	return n
}

// reserveConn counts a connection from ip, unless that would go over a limit.
func (h *Hub) reserveConn(ip string) bool {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	if h.total >= maxConns || h.conns[ip] >= maxConnsPerIP {
		return false
	}
	h.total++
	h.conns[ip]++
	return true
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

// leaveTable removes c from its table, and deletes the table once empty. A
// seated player who disconnected keeps their seat for reconnectGrace.
func (h *Hub) leaveTable(c *Client, disconnected bool) {
	t := c.table
	if t == nil {
		return
	}
	c.table = nil
	select {
	case t.unregister <- c:
	case <-t.done: // closed after its session ended
	}

	t.mu.Lock()
	uuid := c.id()
	// a seated player who dropped keeps their seat, and the host role, for a while
	grace := disconnected && isSeatedUUID(t, uuid)
	if grace {
		t.log(fmt.Sprintf("%s disconnected", c.username))
		t.grace[uuid] = time.AfterFunc(reconnectGrace, func() { h.graceExpired(t, uuid) })
	} else {
		leaveSeat(t, uuid)
	}
	delete(t.users, c)
	if !grace {
		delete(t.requests, c.username)
	}
	if t.host == c.username && !grace {
		t.passHost()
	}
	broadcastGame(t)
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
		name := playerName(t, uuid)
		leaveSeat(t, uuid)
		if name == t.host {
			t.passHost()
			broadcastGame(t)
		}
	}
	t.mu.Unlock()
	h.deleteIfEmpty(t)
}

// deleteIfEmpty deletes t once nobody is connected and no seat is waiting
// for a reconnect. A table whose session ended stays readable until
// closeTable instead.
func (h *Hub) deleteIfEmpty(t *table) {
	h.mu.Lock()
	t.mu.Lock()
	empty := t.members == 0 && len(t.grace) == 0 && h.tables[t.name] == t && t.endedAt.IsZero()
	t.mu.Unlock()
	if empty {
		delete(h.tables, t.name)
	}
	h.mu.Unlock()

	if empty {
		close(t.done)
	}
}

// closeTable deletes a table, telling anyone still there why.
func (h *Hub) closeTable(t *table, reason string) {
	h.mu.Lock()
	open := h.tables[t.name] == t
	if open {
		delete(h.tables, t.name)
	}
	h.mu.Unlock()
	if !open {
		return
	}
	t.mu.Lock()
	for u := range t.users {
		sendErrorCode(u, codeClosed, reason)
	}
	t.mu.Unlock()
	close(t.done)
}

// shutdown closes every table, telling its players the server is restarting,
// then closes every connection once its pending messages are written, or ctx
// is done.
func (h *Hub) shutdown(ctx context.Context) {
	h.mu.Lock()
	tables := slices.Collect(maps.Values(h.tables))
	h.mu.Unlock()
	for _, t := range tables {
		h.closeTable(t, "the server is restarting")
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

// reapInterval is how often idle tables are looked for, a var for tests.
var reapInterval = time.Minute

// reap closes idle tables every reapInterval.
func (h *Hub) reap() {
	for range time.Tick(reapInterval) {
		h.closeIdle(idleTableTimeout)
	}
}

// closeIdle closes the tables no player has acted at for longer than idle.
// Ended sessions close on their own after endedTableTTL.
func (h *Hub) closeIdle(idle time.Duration) {
	h.mu.Lock()
	tables := slices.Collect(maps.Values(h.tables))
	h.mu.Unlock()
	for _, t := range tables {
		t.mu.Lock()
		stale := t.endedAt.IsZero() && time.Since(t.lastActive) > idle
		t.mu.Unlock()
		if stale {
			h.closeTable(t, fmt.Sprintf("the table closed after %s idle", humanDuration(idle)))
		}
	}
}

// humanDuration spells out whole hours or minutes, e.g. "30 minutes".
func humanDuration(d time.Duration) string {
	for _, u := range []struct {
		d    time.Duration
		name string
	}{{time.Hour, "hour"}, {time.Minute, "minute"}} {
		if n := d / u.d; n > 0 && d%u.d == 0 {
			if n == 1 {
				return "1 " + u.name
			}
			return fmt.Sprintf("%d %ss", n, u.name)
		}
	}
	return d.String()
}

// publicTables summarizes the public tables whose session hasn't ended, with
// the most players first and full tables last.
func (h *Hub) publicTables() []tableSummary {
	// settings are fixed after creation, so Public is read without t.mu
	h.mu.Lock()
	var public []*table
	for _, t := range h.tables {
		if t.settings.Public {
			public = append(public, t)
		}
	}
	h.mu.Unlock()

	tables := []tableSummary{}
	for _, t := range public {
		t.mu.Lock()
		if t.endedAt.IsZero() {
			tables = append(tables, t.summary())
		}
		t.mu.Unlock()
	}
	slices.SortFunc(tables, func(a, b tableSummary) int {
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
