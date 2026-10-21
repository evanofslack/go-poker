package server

import (
	"crypto/subtle"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/evanofslack/go-poker/poker"
)

// tableSettings are picked by the host when creating a table, and fixed after.
// Buy-ins are in chips, times in milliseconds.
type tableSettings struct {
	Seats         uint `json:"seats"`
	SmallBlind    uint `json:"sb"`
	BigBlind      uint `json:"bb"`
	MinBuy        uint `json:"minBuy"`
	MaxBuy        uint `json:"maxBuy"`
	TurnTime      uint `json:"turnTime"`      // time to act before checking or folding for you, 0 for no limit
	NextHandDelay uint `json:"nextHandDelay"` // pause after a hand before dealing the next
	Public        bool `json:"public"`        // not used yet, for listing tables later
	ApproveRebuys bool `json:"approveRebuys"` // rebuys, and sitting back down, wait for the host
}

// defaultSettings are used when create-table sends none. A var so tests can
// shorten the timers.
var defaultSettings = tableSettings{
	Seats:         8,
	SmallBlind:    10,
	BigBlind:      20,
	MinBuy:        1000,
	MaxBuy:        4000,
	TurnTime:      30000,
	NextHandDelay: 5000,
}

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
	settings   tableSettings // fixed after creation
	password   string        // empty for none, fixed after creation
	ledger     []ledgerEntry // chips in and out by name, guarded by mu
	hub        *Hub          // closes the table once an ended session expires

	hands     []handStats // the last maxHands hands, guarded by mu
	handCount int         // hands played, guarded by mu
	dealt     []string    // names dealt into this hand, guarded by mu

	requests map[string]buyRequest // buy-ins waiting for the host, by name; guarded by mu

	paused  bool          // no new hands are dealt, guarded by mu
	ending  bool          // the session ends once this hand does, guarded by mu
	endedAt time.Time     // when the session ended, zero until then; guarded by mu
	done    chan struct{} // closed when the table is deleted
	members int           // clients at this table, guarded by Hub.mu

	mu     sync.Mutex       // serializes game actions from client goroutines
	users  map[*Client]bool // clients at this table, guarded by mu
	host   string           // name of the host, guarded by mu
	banker string           // name everyone settles up with, empty for the host; guarded by mu

	dealTimer *time.Timer     // deals the next hand, guarded by mu
	turnTimer *time.Timer     // acts for an idle player, guarded by mu
	turnSeq   int             // invalidates old turn timers, guarded by mu
	leaving   map[string]bool // uuids of players who left, guarded by mu

	turnDeadline time.Time       // when the player to act times out, guarded by mu
	sitOutNext   map[string]bool // uuids of players sitting out after this hand, guarded by mu

	// seated players who disconnect can reconnect within reconnectGrace.
	// Both guarded by mu.
	tokens map[string]string      // session token to player uuid
	grace  map[string]*time.Timer // player uuid to the timer that removes them

	bots map[string]botSeat // bot players by uuid, guarded by mu
	rng  *rand.Rand         // for bot moves, guarded by mu
	// botDelay is copied from the global so a test can change one table's
	botDelay func() time.Duration
}

// newTable creates a new table
func newTable(name string, settings tableSettings) *table {
	return &table{
		name:       name,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		updates:    make(chan stateUpdate),
		game: poker.NewGame(poker.GameConfig{
			MaxBuy:     settings.MaxBuy,
			BigBlind:   settings.BigBlind,
			SmallBlind: settings.SmallBlind,
		}),
		settings:   settings,
		done:       make(chan struct{}),
		users:      make(map[*Client]bool),
		leaving:    make(map[string]bool),
		sitOutNext: make(map[string]bool),
		tokens:     make(map[string]string),
		grace:      make(map[string]*time.Timer),
		bots:       make(map[string]botSeat),
		rng:        rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		botDelay:   botDelay,
		requests:   make(map[string]buyRequest),
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

// checkPassword reports why password doesn't let someone join, if it doesn't.
func (t *table) checkPassword(password string) error {
	switch {
	case t.password == "":
		return nil
	case password == "":
		return errNeedPassword
	case subtle.ConstantTimeCompare([]byte(password), []byte(t.password)) != 1:
		return errWrongPassword
	}
	return nil
}

// reclaim returns the uuid and name of the player that token belongs to, if
// they're still seated. Caller holds t.mu.
func (t *table) reclaim(token string) (uuid string, name string, ok bool) {
	uuid, ok = t.tokens[token]
	if !ok || t.leaving[uuid] {
		return "", "", false
	}
	for _, p := range t.game.GenerateOmniView().Players {
		if p.UUID == uuid {
			return uuid, p.Username, true
		}
	}
	return "", "", false
}

// ended reports whether the session has ended. It takes t.mu.
func (t *table) ended() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return !t.endedAt.IsZero()
}

// closesIn is how many milliseconds an ended table has left, or 0 if the
// session hasn't ended. Caller holds t.mu.
func (t *table) closesIn() int64 {
	if t.endedAt.IsZero() {
		return 0
	}
	return max(time.Until(t.endedAt.Add(endedTableTTL)).Milliseconds(), 0)
}

// passHost gives the host role to a connected seated player, else to whoever
// is at the table first by name, else nobody. Caller holds t.mu.
func (t *table) passHost() {
	t.host = ""
	for _, p := range t.game.GenerateOmniView().Players {
		if t.connected(p.Username) && !t.leaving[p.UUID] {
			t.host = p.Username
			return
		}
	}
	for u := range t.users {
		if t.host == "" || u.username < t.host {
			t.host = u.username
		}
	}
}

// connected reports whether a client named name is at the table. Caller holds t.mu.
func (t *table) connected(name string) bool {
	for u := range t.users {
		if u.username == name {
			return true
		}
	}
	return false
}

// watchers are the names of the clients at the table without a seat, sorted.
// Caller holds t.mu.
func (t *table) watchers() []string {
	names := []string{}
	for u := range t.users {
		if !isSeatedUUID(t, u.id()) {
			names = append(names, u.username)
		}
	}
	slices.Sort(names)
	return names
}

// disconnected are the names of seated players waiting out their reconnect
// grace period, sorted. Caller holds t.mu.
func (t *table) disconnected() []string {
	names := []string{}
	for _, p := range t.game.GenerateOmniView().Players {
		if _, ok := t.grace[p.UUID]; ok {
			names = append(names, p.Username)
		}
	}
	slices.Sort(names)
	return names
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
