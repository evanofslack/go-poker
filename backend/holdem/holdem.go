// Package holdem is a no-limit Texas hold'em engine. State is plain data: no
// locks, no clock and no global randomness, so it can be copied, compared and
// saved. Methods check an action is legal before changing anything, so one that
// returns an error leaves the state as it was.
package holdem

import (
	"errors"
	"slices"
	"sort"

	"github.com/alexclewontin/riverboat/eval"
)

// Card is a playing card, as the evaluator encodes it.
type Card = eval.Card

// Street is a betting round. NoStreet is between hands.
type Street string

const (
	NoStreet Street = ""
	Preflop  Street = "preflop"
	Flop     Street = "flop"
	Turn     Street = "turn"
	River    Street = "river"
)

var (
	ErrNotYourTurn      = errors.New("not your turn")
	ErrIllegal          = errors.New("that isn't allowed right now")
	ErrNotEnoughPlayers = errors.New("need at least two players")
	ErrInHand           = errors.New("wait for the hand to end")
	ErrSeatTaken        = errors.New("seat taken")
	ErrNoSeat           = errors.New("nobody in that seat")
	ErrBadAmount        = errors.New("invalid amount")
)

// Config is fixed for a game.
type Config struct {
	SmallBlind uint `json:"sb"`
	BigBlind   uint `json:"bb"`
	Ante       uint `json:"ante"` // from every player dealt in, 0 for none
}

// Player is someone in a seat.
type Player struct {
	Seat       int     `json:"seat"` // from 1
	Stack      uint    `json:"stack"`
	Bet        uint    `json:"bet"`        // this street
	Committed  uint    `json:"committed"`  // this hand, Bet included
	Dealt      bool    `json:"dealt"`      // dealt into the last or current hand
	In         bool    `json:"in"`         // dealt in and not folded
	SittingOut bool    `json:"sittingOut"` // not dealt in; busted players sit out
	Acted      bool    `json:"acted"`      // since the last full raise, so a short all-in can't be re-raised
	Cards      [2]Card `json:"cards"`      // zero when not dealt, or hidden in a view
	Shown      bool    `json:"shown"`      // face up: once all in, at showdown if they must, or by choice after
}

// allIn reports whether p is in the hand with no chips left to bet.
func (p *Player) allIn() bool { return p.In && p.Stack == 0 }

// canBet reports whether p is in the hand with chips left to bet.
func (p *Player) canBet() bool { return p.In && p.Stack > 0 }

// ready reports whether p will be dealt into the next hand.
func (p *Player) ready() bool { return !p.SittingOut && p.Stack > 0 }

// put moves up to amt chips from p's stack into their bet, and returns how many.
func (p *Player) put(amt uint) uint {
	amt = min(amt, p.Stack)
	p.Stack -= amt
	p.Bet += amt
	p.Committed += amt
	return amt
}

// putDead moves up to amt chips from p's stack into the pot without betting
// them, like an ante, and returns how many.
func (p *Player) putDead(amt uint) uint {
	amt = min(amt, p.Stack)
	p.Stack -= amt
	p.Committed += amt
	return amt
}

// Pot is a main or side pot. Between hands it holds the last hand's result.
type Pot struct {
	Amount   uint   `json:"amount"`
	Eligible []int  `json:"eligible"` // seats that can win it
	Winners  []int  `json:"winners"`
	Cards    []Card `json:"cards"` // the winning five, empty without a showdown
	Hand     string `json:"hand"`  // like "two pair, kings and sevens", empty without a showdown
	Score    int    `json:"score"` // of the winning hand, lower is better; 0 without a showdown
}

// State is a game of hold'em.
type State struct {
	Config    Config   `json:"config"`
	Players   []Player `json:"players"` // by seat
	Street    Street   `json:"street"`
	Betting   bool     `json:"betting"` // someone has to act; false mid-hand means NextStreet is due
	Button    int      `json:"button"`  // seats, 0 for none
	SB        int      `json:"sb"`
	BB        int      `json:"bb"`
	ToAct     int      `json:"toAct"`
	Board     []Card   `json:"board"`
	Hunted    []Card   `json:"hunted"` // the rest of the board, turned up for fun after a hand ended early
	Deck      []Card   `json:"deck"`   // kept after the hand for Hunt
	Pots      []Pot    `json:"pots"`
	MinRaise  uint     `json:"minRaise"`  // the smallest raise, by how much it adds to the top bet
	Aggressor int      `json:"aggressor"` // the last to bet or raise this street, 0 for none
	Hands     int      `json:"hands"`     // dealt so far
	Seed      [32]byte `json:"seed"`      // decks come from this and the hand number
}

// New returns a game with no players. Each hand's deck is shuffled from seed
// and the hand number, so a seed replays the same cards.
func New(cfg Config, seed [32]byte) State {
	return State{Config: cfg, Seed: seed}
}

// Clone returns a deep copy of s.
func (s State) Clone() State {
	s.Players = slices.Clone(s.Players)
	s.Board = slices.Clone(s.Board)
	s.Hunted = slices.Clone(s.Hunted)
	s.Deck = slices.Clone(s.Deck)
	pots := make([]Pot, len(s.Pots))
	for i, p := range s.Pots {
		p.Eligible = slices.Clone(p.Eligible)
		p.Winners = slices.Clone(p.Winners)
		p.Cards = slices.Clone(p.Cards)
		pots[i] = p
	}
	if s.Pots == nil {
		pots = nil
	}
	s.Pots = pots
	return s
}

// Player returns the player in seat, or nil.
func (s *State) Player(seat int) *Player {
	for i := range s.Players {
		if s.Players[i].Seat == seat {
			return &s.Players[i]
		}
	}
	return nil
}

// InHand reports whether a hand is being played.
func (s *State) InHand() bool { return s.Street != NoStreet }

// next is the first seat after seat, going round the table, whose player
// matches ok, or 0. seat itself comes last.
func (s *State) next(seat int, ok func(*Player) bool) int {
	n := len(s.Players)
	start := sort.Search(n, func(i int) bool { return s.Players[i].Seat > seat })
	for k := range n {
		p := &s.Players[(start+k)%n]
		if ok(p) {
			return p.Seat
		}
	}
	return 0
}

// count is how many players match ok.
func (s *State) count(ok func(*Player) bool) int {
	n := 0
	for i := range s.Players {
		if ok(&s.Players[i]) {
			n++
		}
	}
	return n
}

// topBet is the biggest bet this street.
func (s *State) topBet() uint {
	var top uint
	for _, p := range s.Players {
		top = max(top, p.Bet)
	}
	return top
}

// ReadyCount is how many players would be dealt into a hand starting now.
func (s *State) ReadyCount() int {
	return s.count((*Player).ready)
}
