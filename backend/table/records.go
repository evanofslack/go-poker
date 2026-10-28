package table

import (
	"slices"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// MaxHands caps how many hands of history and stats a table keeps in memory.
const MaxHands = 1000

// handsPerPage is how many hands of history go out at once.
const handsPerPage = 20

// HandRecord is one hand as it was played. Records keep every player's hole
// cards, so they only leave the table through viewFor.
type HandRecord struct {
	Hand    int          `json:"hand"`    // counts from 1
	Started int64        `json:"started"` // unix ms
	SB      uint         `json:"sb"`
	BB      uint         `json:"bb"`
	Ante    uint         `json:"ante,omitempty"`
	Dealer  string       `json:"dealer"`
	Players []HandPlayer `json:"players"` // dealt in, in seat order
	Actions []HandAction `json:"actions"` // antes and blinds first
	Board   []string     `json:"board"`
	Boards  [][]string   `json:"boards,omitempty"` // every board, when it ran out more than once
	Pots    []HandPot    `json:"pots"`
	// Payments are side games paid out after the hand
	Payments []Payment `json:"payments,omitempty"`
}

type HandPlayer struct {
	Name     string   `json:"name"`
	Seat     int      `json:"seat"`
	Stack    uint     `json:"stack"`           // before the antes and blinds
	Cards    []string `json:"cards,omitempty"` // hidden from others unless Shown
	Shown    bool     `json:"shown"`           // face up: at showdown if they had to, or by choice
	Hand     string   `json:"hand,omitempty"`  // their best hand, if Shown
	Showdown bool     `json:"showdown"`        // reached showdown, shown or mucked
	Net      int      `json:"net"`             // chips won or lost
	Score    int      `json:"-"`               // of Hand, lower is better
}

type HandAction struct {
	Street  string `json:"street"` // preflop, flop, turn or river
	Name    string `json:"name"`
	Kind    string `json:"kind"`             // ante, small blind, big blind, straddle, check, call, bet, raise or fold
	Amount  uint   `json:"amount,omitempty"` // chips put in
	To      uint   `json:"to,omitempty"`     // their bet on this street after putting them in
	AllIn   bool   `json:"allIn,omitempty"`
	Timeout bool   `json:"timeout,omitempty"` // the turn timer acted for them
}

type HandPot struct {
	Amount  uint     `json:"amount"`
	Winners []string `json:"winners"`
	Hand    string   `json:"hand,omitempty"` // empty when won without a showdown
	Run     int      `json:"run,omitempty"`  // which board it was won on, from 0, when it ran out more than once
}

// HandStats is a snapshot taken as a hand ends. Stats are worked out from
// these on the client.
type HandStats struct {
	Hand    int             `json:"hand"`    // counts from 1
	Started int64           `json:"started"` // unix ms
	Stacks  map[string]uint `json:"stacks"`  // everyone seated, after the pots are paid
	Played  []string        `json:"played"`  // names dealt in
	Won     map[string]uint `json:"won"`     // pots each winner took, their own chips included
}

// HandDone is a hand finishing: its record, as the receiver may see it, its
// stats, and everyone's totals.
type HandDone struct {
	Record HandRecord        `json:"record"`
	Stats  HandStats         `json:"stats"`
	Totals map[string]Totals `json:"totals"`
}

// HandUpdated is the last hand's record changing after it ended, when
// someone shows their cards.
type HandUpdated struct {
	Record HandRecord `json:"record"`
}

// Log is a line in the table's log.
type Log struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
	Time    int64  `json:"time"` // unix ms
}

// Logged is a new line in the table's log.
type Logged struct {
	Log Log `json:"log"`
}

func (HandDone) EventType() string    { return "hand-done" }
func (HandUpdated) EventType() string { return "hand-updated" }
func (Logged) EventType() string      { return "log" }

// Event is something that happened at the table: a holdem.Event, HandDone,
// HandUpdated or Logged.
type Event = holdem.Event

func cardNames(cards ...holdem.Card) []string {
	var names []string
	for _, c := range cards {
		if c != 0 {
			names = append(names, c.String())
		}
	}
	return names
}

// recordEvent adds what the game did to the hand being recorded, and to the
// actions the seats show.
func (r *reducer) recordEvent(e holdem.Event, timeout bool) {
	s := r.s
	h := s.Current
	name := func(seat int) string { return s.Seats[seat].Name }
	if _, ok := e.(holdem.HandStarted); !ok && h == nil {
		return
	}

	switch e := e.(type) {
	case holdem.HandStarted:
		h = &HandRecord{
			Hand:    e.Hand,
			Started: r.now.UnixMilli(),
			SB:      s.Game.Config.SmallBlind,
			BB:      s.Game.Config.BigBlind,
			Ante:    s.Game.Config.Ante,
			Dealer:  name(e.Button),
		}
		for _, n := range e.Seats {
			p := s.Game.Player(n)
			h.Players = append(h.Players, HandPlayer{Name: name(n), Seat: n, Stack: p.Stack + p.Committed})
		}
		s.Current = h
		r.beginStandUp(e.Seats)
	case holdem.Dealt:
		if hp := h.player(e.Seat); hp != nil {
			hp.Cards = cardNames(e.Cards[:]...)
		}
	case holdem.Acted:
		a := HandAction{Street: string(e.Street), Name: name(e.Seat), Kind: e.Kind, Amount: e.Amount, AllIn: e.AllIn, Timeout: timeout}
		if e.Amount > 0 {
			a.To = e.To
		}
		h.Actions = append(h.Actions, a)
		if e.Kind != holdem.KindAnte && e.Kind != holdem.KindSmallBlind && e.Kind != holdem.KindBigBlind && e.Kind != holdem.KindStraddle {
			s.Acts[e.Seat] = Act{e.Street, actLabel(e)}
		}
	case holdem.StreetDealt:
		if e.Run == 0 {
			h.Board = append(h.Board, cardNames(e.Cards...)...)
		}
	case holdem.Shown:
		if hp := h.player(e.Seat); hp != nil {
			hp.Shown = true
		}
	case holdem.PotWon:
		if e.Amount == 0 || len(e.Winners) == 0 {
			break
		}
		pot := HandPot{Amount: e.Amount, Hand: e.Hand, Run: e.Run}
		for _, n := range e.Winners {
			pot.Winners = append(pot.Winners, name(n))
		}
		h.Pots = append(h.Pots, pot)
	case holdem.HandEnded:
		r.finishHand()
	}
}

func actLabel(e holdem.Acted) string {
	switch {
	case e.AllIn:
		return "all in"
	case e.Kind == holdem.KindCall:
		return "call " + itoa(e.Amount)
	case e.Kind == holdem.KindBet:
		return "bet " + itoa(e.Amount)
	case e.Kind == holdem.KindRaise:
		return "raise to " + itoa(e.To)
	}
	return e.Kind
}

// aggressor is the seat that made the last bet or raise before street, 0 if
// nobody did. Actions are in order, so the first on street ends the search.
func (h *HandRecord) aggressor(street holdem.Street) int {
	seat := 0
	if h == nil {
		return seat
	}
	for _, a := range h.Actions {
		if a.Street == string(street) {
			break
		}
		if a.Kind == holdem.KindBet || a.Kind == holdem.KindRaise {
			for _, p := range h.Players {
				if p.Name == a.Name {
					seat = p.Seat
				}
			}
		}
	}
	return seat
}

func (h *HandRecord) player(seat int) *HandPlayer {
	for i := range h.Players {
		if h.Players[i].Seat == seat {
			return &h.Players[i]
		}
	}
	return nil
}

// finishHand completes the hand that just ended, adds it to the stats and
// totals, keeps it and tells everyone.
func (r *reducer) finishHand() {
	s := r.s
	h := s.Current
	s.Current = nil
	if h == nil {
		return
	}

	// everyone who could win a pot that was scored reached showdown
	showdown := map[int]bool{}
	for _, pot := range s.Game.Pots {
		for _, n := range pot.Eligible {
			showdown[n] = showdown[n] || len(pot.Runs) > 0
		}
	}
	// side games pay before the nets, so they count in them
	for _, hp := range h.Players {
		h.Payments = append(h.Payments, r.sevenDeuce(h, hp.Seat)...)
	}
	h.Payments = append(h.Payments, r.standUp()...)
	for i := range h.Players {
		hp := &h.Players[i]
		p := s.Game.Player(hp.Seat)
		hp.Net = int(p.Stack) - int(hp.Stack)
		hp.Showdown = showdown[hp.Seat]
		if hp.Shown {
			hp.bestHand(p.Cards, s.Game.Board)
		}
	}
	if len(s.Game.Boards) > 1 {
		for _, b := range s.Game.Boards {
			h.Boards = append(h.Boards, cardNames(b...))
		}
	}
	tally(s.Totals, h)
	s.History = append(s.History, *h)
	if len(s.History) > MaxHands {
		s.History = s.History[len(s.History)-MaxHands:]
	}

	st := HandStats{Hand: h.Hand, Started: h.Started, Stacks: map[string]uint{}, Played: h.played(), Won: map[string]uint{}}
	for _, p := range s.Game.Players {
		st.Stacks[s.Seats[p.Seat].Name] = p.Stack
	}
	for _, pot := range h.Pots {
		// odd chips aren't tracked; this is close enough for stats
		for _, w := range pot.Winners {
			st.Won[w] += pot.Amount / uint(len(pot.Winners))
		}
	}
	s.Stats = append(s.Stats, st)
	if len(s.Stats) > MaxHands {
		s.Stats = s.Stats[len(s.Stats)-MaxHands:]
	}
	r.res.Events = append(r.res.Events, HandDone{Record: *h, Stats: st, Totals: TotalsView(s)})
}

// bestHand names hp's best hand, with a full board.
func (hp *HandPlayer) bestHand(c [2]holdem.Card, board []holdem.Card) {
	if len(board) == 5 {
		cards, score := eval.BestFiveOfSeven(c[0], c[1], board[0], board[1], board[2], board[3], board[4])
		hp.Hand, hp.Score = holdem.HandName(score, cards), score
	}
}

// played names everyone dealt into h.
func (h *HandRecord) played() []string {
	var names []string
	for _, p := range h.Players {
		names = append(names, p.Name)
	}
	return names
}

// viewFor is h as name may see it: their own cards and any shown at
// showdown. Spectators pass "".
func (h HandRecord) viewFor(name string) HandRecord {
	h.Players = slices.Clone(h.Players)
	for i, p := range h.Players {
		if !p.Shown && (name == "" || p.Name != name) {
			h.Players[i].Cards = nil
			h.Players[i].Hand = ""
		}
	}
	return h
}

// HistoryFor is up to handsPerPage hands from before hand number before (0
// for the latest), as name may see them, and whether there are older ones.
func HistoryFor(s *State, name string, before int) ([]HandRecord, bool) {
	end := len(s.History)
	if before > 0 {
		end = 0
		for end < len(s.History) && s.History[end].Hand < before {
			end++
		}
	}
	start := max(end-handsPerPage, 0)
	hands := make([]HandRecord, 0, end-start)
	for _, h := range s.History[start:end] {
		hands = append(hands, h.viewFor(name))
	}
	return hands, start > 0
}
