// Package table is a poker table as plain data: who is there, the game, the
// ledger, history and timers. Reduce applies a command at a given time and
// says what to send; it does no I/O and reads no clock, so a table can be
// tested with a fake clock and saved by serializing its State.
package table

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	mrand "math/rand/v2"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// Version is the version of State, for loading saved tables.
const Version = 1

// Timing is how long the table waits for things. Tests shorten it.
type Timing struct {
	ReconnectGrace time.Duration `json:"reconnectGrace"` // a disconnected player keeps their seat
	EndedTTL       time.Duration `json:"endedTTL"`       // an ended table stays readable
	IdleTimeout    time.Duration `json:"idleTimeout"`    // a table without player messages closes
	StreetDelay    time.Duration `json:"streetDelay"`    // between a street's last action and the next street
	BotDelay       time.Duration `json:"botDelay"`       // a bot thinks for this long, plus up to as long again
}

var DefaultTiming = Timing{
	ReconnectGrace: 60 * time.Second,
	EndedTTL:       time.Hour,
	IdleTimeout:    30 * time.Minute,
	StreetDelay:    time.Second,
	BotDelay:       time.Second,
}

// Person is someone at the table: connected, or seated and waiting out their
// reconnect grace.
type Person struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Seat      int    `json:"seat"` // 0 when watching
	Connected bool   `json:"connected"`
}

// Seat is the table's side of a player in the game.
type Seat struct {
	Name       string `json:"name"`
	Person     string `json:"person"`   // id of who sits here; empty for a bot, or once they left
	Bot        string `json:"bot"`      // level, empty for a person
	BotBuyIn   uint   `json:"botBuyIn"` // what a bot rebuys for
	Token      string `json:"token"`    // hash of the token that reclaims the seat
	Leaving    bool   `json:"leaving"`  // folded on their turn and removed after the hand
	SitOutNext bool   `json:"sitOutNext"`
}

// Timer names a deadline. The runtime arms one timer per entry in
// State.Timers and sends TimerFired when it's due.
type Timer string

const (
	TimerDeal   Timer = "deal"   // deal the next hand
	TimerStreet Timer = "street" // deal the next street
	TimerTurn   Timer = "turn"   // act for the player to act
	TimerBot    Timer = "bot"    // a bot to act moves
	TimerClose  Timer = "close"  // an ended table closes
	TimerIdle   Timer = "idle"   // the table closes for lack of players
)

// graceTimer removes a disconnected person who didn't come back.
func graceTimer(id string) Timer { return Timer("grace:" + id) }

// Act is what a seat last did, which it shows until the street changes.
type Act struct {
	Street holdem.Street `json:"street"`
	Label  string        `json:"label"` // like "check" or "raise to 80"
}

// State is a table. Everything in it is serializable. History, Stats and
// Logs only grow, so they're left out of the JSON and saved as they're added.
type State struct {
	Version   int          `json:"version"`
	ID        string       `json:"id"` // names are reused once a table closes; IDs aren't
	Name      string       `json:"name"`
	Settings  Settings     `json:"settings"`
	Timing    Timing       `json:"timing"`
	Password  string       `json:"password"` // salted hash, empty for none
	CreatorIP string       `json:"creatorIP"`
	Game      holdem.State `json:"game"`

	Running bool      `json:"running"` // hands are dealt
	Paused  bool      `json:"paused"`  // no new hands
	Ending  bool      `json:"ending"`  // the session ends after this hand
	EndedAt time.Time `json:"endedAt"` // zero until the session ends

	People map[string]*Person `json:"people"` // by id
	Seats  map[int]*Seat      `json:"seats"`  // by seat number
	Host   string             `json:"host"`   // name
	Banker string             `json:"banker"` // name, empty for the host

	Ledger   []LedgerEntry         `json:"ledger"`
	Requests map[string]BuyRequest `json:"requests"` // by name
	History  []HandRecord          `json:"-"`        // the last MaxHands hands
	Current  *HandRecord           `json:"current"`  // the hand being played
	Stats    []HandStats           `json:"-"`        // the last MaxHands hands, alongside History
	Totals   map[string]*Totals    `json:"totals"`   // by name
	Logs     []Log                 `json:"-"`        // the last MaxLogs lines
	LogSeq   int                   `json:"logSeq"`
	Acts     map[int]Act           `json:"acts"` // by seat

	Timers  map[Timer]time.Time `json:"timers"`
	RandSeq uint64              `json:"randSeq"` // draws from the table's randomness so far
}

// New creates a table. The password is kept hashed.
func New(name string, settings Settings, timing Timing, password, creatorIP string, now time.Time) *State {
	var seed [32]byte
	rand.Read(seed[:])
	s := &State{
		Version:   Version,
		ID:        newID(),
		Name:      name,
		Settings:  settings,
		Timing:    timing,
		CreatorIP: creatorIP,
		Game:      holdem.New(holdem.Config{SmallBlind: settings.SmallBlind, BigBlind: settings.BigBlind, Ante: settings.Ante}, seed),
		People:    map[string]*Person{},
		Seats:     map[int]*Seat{},
		Requests:  map[string]BuyRequest{},
		Totals:    map[string]*Totals{},
		Acts:      map[int]Act{},
		Timers:    map[Timer]time.Time{TimerIdle: now.Add(timing.IdleTimeout)},
	}
	if password != "" {
		s.Password = hashPassword(password)
	}
	return s
}

// Ended reports whether the session is over.
func (s *State) Ended() bool { return !s.EndedAt.IsZero() }

// rand is the table's next source of randomness, from the game seed. It
// makes bot play replay from a saved table.
func (s *State) rand() *mrand.Rand {
	s.RandSeq++
	return mrand.New(mrand.NewPCG(binary.LittleEndian.Uint64(s.Game.Seed[:8]), s.RandSeq))
}

// person finds who is called name, ignoring case.
func (s *State) personNamed(name string) *Person {
	for _, p := range s.People {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	return nil
}

// seatNamed is the seat held by name and not being left, or 0.
func (s *State) seatNamed(name string) int {
	for n, seat := range s.Seats {
		if seat.Name == name && !seat.Leaving {
			return n
		}
	}
	return 0
}

// nameTaken reports whether name is used by anyone at the table, seated or
// not, ignoring case.
func (s *State) nameTaken(name string) bool {
	if s.personNamed(name) != nil {
		return true
	}
	for _, seat := range s.Seats {
		if strings.EqualFold(seat.Name, name) {
			return true
		}
	}
	return false
}

// connected reports whether a person called name is connected.
func (s *State) connected(name string) bool {
	p := s.personNamed(name)
	return p != nil && p.Name == name && p.Connected
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	sum := sha256.Sum256(append(salt, password...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(sum[:])
}

// checkPassword reports whether password matches the hash from hashPassword.
func checkPassword(hash, password string) bool {
	saltHex, sumHex, ok := strings.Cut(hash, ":")
	salt, err := hex.DecodeString(saltHex)
	if !ok || err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, password...))
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(sumHex)) == 1
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// newToken is a secret for reclaiming a seat, and the hash the table keeps.
func newToken() (token, hash string) {
	b := make([]byte, 16)
	rand.Read(b)
	token = hex.EncodeToString(b)
	return token, hashToken(token)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Restore readies a table loaded from storage after a restart. Nobody is
// connected anymore: seated people get the reconnect grace and watchers are
// gone. The player to act gets a fresh turn.
func Restore(s *State, now time.Time) {
	for id, p := range s.People {
		if p.Seat == 0 {
			delete(s.People, id)
			continue
		}
		p.Connected = false
		s.Timers[graceTimer(id)] = now.Add(s.Timing.ReconnectGrace)
	}
	if _, ok := s.Timers[TimerTurn]; ok {
		s.Timers[TimerTurn] = now.Add(time.Duration(s.Settings.TurnTime) * time.Millisecond)
	}
}
