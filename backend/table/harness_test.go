package table

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

var fast = Timing{
	ReconnectGrace: time.Minute,
	EndedTTL:       time.Hour,
	IdleTimeout:    30 * time.Minute,
	StreetDelay:    time.Second,
	BotDelay:       time.Second,
}

// harness drives a table with a fake clock.
type harness struct {
	t      *testing.T
	s      *State
	now    time.Time
	out    []Out
	events []Event
	closed *Close
	ids    map[string]string // person id by name
	tokens map[string]string // seat token by name
}

func newHarness(t *testing.T, settings Settings, password string) *harness {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return &harness{
		t:      t,
		s:      New("t", settings, fast, password, "1.2.3.4", now),
		now:    now,
		ids:    map[string]string{},
		tokens: map[string]string{},
	}
}

// do applies cmd and keeps what it said.
func (h *harness) do(cmd Command) Result {
	h.t.Helper()
	res := Reduce(h.s, h.now, cmd)
	h.out = append(h.out, res.Out...)
	h.events = append(h.events, res.Events...)
	if res.Close != nil {
		h.closed = res.Close
	}
	for _, o := range res.Out {
		if tok, ok := o.Msg.(NewToken); ok {
			for name, id := range h.ids {
				if id == o.To {
					h.tokens[name] = tok.Token
				}
			}
		}
	}
	return res
}

// ok applies cmd and fails if it was refused.
func (h *harness) ok(cmd Command) Result {
	h.t.Helper()
	res := h.do(cmd)
	for _, o := range res.Out {
		if e, isErr := o.Msg.(Error); isErr && o.To == PersonOf(cmd) && e.Code == "" {
			h.t.Fatalf("%T refused: %s", cmd, e.Message)
		}
	}
	return res
}

// refused applies cmd and returns the error it got, failing if none.
func (h *harness) refused(cmd Command) string {
	h.t.Helper()
	res := h.do(cmd)
	for _, o := range res.Out {
		if e, isErr := o.Msg.(Error); isErr {
			return e.Message
		}
	}
	h.t.Fatalf("%T wasn't refused", cmd)
	return ""
}

func (h *harness) join(name string) string {
	h.t.Helper()
	id := "id-" + name
	h.ids[name] = id
	res := h.do(Join{ID: id, Name: name})
	if res.Joined == nil {
		h.t.Fatalf("%s couldn't join: %+v", name, res.Out)
	}
	return id
}

func (h *harness) sit(name string, seat int, buyIn uint) {
	h.t.Helper()
	h.ok(TakeSeat{Person: h.ids[name], Seat: seat, BuyIn: buyIn})
}

// advance moves the clock on by d, firing the timers that come due in order.
func (h *harness) advance(d time.Duration) {
	h.t.Helper()
	end := h.now.Add(d)
	for range 10000 {
		timer, at, ok := h.nextTimer()
		if !ok || at.After(end) || h.closed != nil {
			h.now = end
			return
		}
		h.now = at
		h.do(TimerFired{timer, at})
	}
	h.t.Fatal("timers never settled")
}

func (h *harness) nextTimer() (Timer, time.Time, bool) {
	var first Timer
	var at time.Time
	for timer, t := range h.s.Timers {
		if first == "" || t.Before(at) || (t.Equal(at) && timer < first) {
			first, at = timer, t
		}
	}
	return first, at, first != ""
}

// toAct is the name of the player to act.
func (h *harness) toAct() string {
	if !h.s.Game.Betting {
		h.t.Fatal("nobody to act")
	}
	return h.s.Seats[h.s.Game.ToAct].Name
}

// play has the player to act make move a.
func (h *harness) play(a holdem.Action) {
	h.t.Helper()
	h.ok(Play{Person: h.ids[h.toAct()], Action: a})
}

// logged reports whether a log line containing text was written.
func (h *harness) logged(text string) bool {
	for _, l := range h.s.Logs {
		if strings.Contains(l.Message, text) {
			return true
		}
	}
	return false
}

func (h *harness) mustLog(text string) {
	h.t.Helper()
	if !h.logged(text) {
		var lines []string
		for _, l := range h.s.Logs {
			lines = append(lines, l.Message)
		}
		h.t.Fatalf("no log line with %q in:\n%s", text, strings.Join(lines, "\n"))
	}
}

// sentTo are the messages sent to id, everyone's included, of type T.
func sentTo[T any](h *harness, id string) []T {
	var got []T
	for _, o := range h.out {
		if m, ok := o.Msg.(T); ok && (o.To == id || o.To == "") {
			got = append(got, m)
		}
	}
	return got
}

// stack is name's chips, counting bets this hand.
func (h *harness) stack(name string) uint {
	for n, seat := range h.s.Seats {
		if seat.Name == name {
			p := h.s.Game.Player(n)
			return p.Stack + p.Committed
		}
	}
	return 0
}

// checkInvariants fails if the table contradicts itself.
func (h *harness) checkInvariants() {
	h.t.Helper()
	s := h.s
	if b := Ledger(s).Balance; b != 0 {
		h.t.Fatalf("ledger off by %d", b)
	}
	for id, p := range s.People {
		if p.ID != id {
			h.t.Fatalf("person %s filed under %s", p.ID, id)
		}
		if p.Seat != 0 && (s.Seats[p.Seat] == nil || s.Seats[p.Seat].Person != id) {
			h.t.Fatalf("%s thinks they're in seat %d", p.Name, p.Seat)
		}
		if !p.Connected && p.Seat == 0 {
			h.t.Fatalf("%s is disconnected without a seat to wait for", p.Name)
		}
		if _, ok := s.Timers[graceTimer(id)]; ok == p.Connected {
			h.t.Fatalf("%s connected %v but grace timer %v", p.Name, p.Connected, ok)
		}
	}
	for n, seat := range s.Seats {
		if s.Game.Player(n) == nil {
			h.t.Fatalf("seat %d (%s) isn't in the game", n, seat.Name)
		}
		if seat.Person != "" && (s.People[seat.Person] == nil || s.People[seat.Person].Seat != n) {
			h.t.Fatalf("seat %d points at %q who isn't there", n, seat.Person)
		}
	}
	if len(s.Seats) != len(s.Game.Players) {
		h.t.Fatalf("%d seats, %d players", len(s.Seats), len(s.Game.Players))
	}
	g := &s.Game
	_, turn := s.Timers[TimerTurn]
	if g.Betting && s.Settings.TurnTime > 0 && !turn {
		h.t.Fatal("betting without a turn timer")
	}
	if !g.Betting && turn {
		h.t.Fatal("turn timer while nobody is betting")
	}
	if _, ok := s.Timers[TimerStreet]; ok != (g.InHand() && !g.Betting && s.RunVote == nil) {
		h.t.Fatalf("street timer %v, in hand %v, betting %v, run vote %+v", ok, g.InHand(), g.Betting, s.RunVote)
	}
	if _, ok := s.Timers[TimerRunVote]; ok != (s.RunVote != nil) {
		h.t.Fatalf("run vote timer %v, vote %+v", ok, s.RunVote)
	}
	if v := s.RunVote; v != nil && (g.Betting || !g.InHand() || g.Runs != 0 || len(v.Seats) < 2) {
		h.t.Fatalf("run vote %+v, betting %v, in hand %v, runs %d", v, g.Betting, g.InHand(), g.Runs)
	}
	if g.Betting && s.Seats[g.ToAct].Bot != "" {
		if _, ok := s.Timers[TimerBot]; !ok {
			h.t.Fatal("a bot to act without a bot timer")
		}
	}
	a := s.StraddleAsk
	if _, ok := s.Timers[TimerStraddle]; ok != (a != nil && a.Seat != 0) {
		h.t.Fatalf("straddle timer %v, ask %+v", ok, a)
	}
	if a != nil && (g.InHand() || !slices.Equal(g.Positions().Straddle[:len(a.Yes)], a.Yes)) {
		h.t.Fatalf("ask %+v in hand %v, could straddle %v", a, g.InHand(), g.Positions().Straddle)
	}
	if s.Next != nil && !g.InHand() {
		h.t.Fatal("settings left waiting between hands")
	}
	if s.Game.Config != s.Settings.config() {
		h.t.Fatalf("game config %+v doesn't match the settings", s.Game.Config)
	}
	if s.Ended() && (g.InHand() || len(s.Requests) > 0) {
		h.t.Fatal("ended mid-hand or with requests")
	}
	if !slices.IsSortedFunc(BuyRequests(s), func(a, b BuyRequest) int { return a.At.Compare(b.At) }) {
		h.t.Fatal("requests out of order")
	}
}

func (h *harness) String() string {
	return fmt.Sprintf("street %q betting %v to act %d", h.s.Game.Street, h.s.Game.Betting, h.s.Game.ToAct)
}
