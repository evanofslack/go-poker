// Package bot picks moves for computer players. It only reads a game view, so
// it can't change the game or see more than the view shows.
package bot

import (
	"math/rand/v2"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// bot levels
const (
	Easy   = "easy"
	Medium = "medium"
	Hard   = "hard"
)

// Kind is the kind of move a bot makes.
type Kind int

const (
	Check Kind = iota
	Call
	Raise
	Fold
)

// Action is a bot's move before it becomes a holdem.Action. Amount is the
// chips it puts in, for Raise only. Reason says why the bot chose it, for logs.
type Action struct {
	Kind   Kind
	Amount uint
	Reason string
}

// Valid reports whether level is a bot level.
func Valid(level string) bool {
	return level == Easy || level == Medium || level == Hard
}

// Read is what everyone at the table has seen of a player over the session,
// like the table's totals.
type Read struct {
	Hands         int // dealt in
	Bombs         int // of them bomb pots, with no preflop to count
	VPIP          int // hands they chose to put money in preflop
	PFR           int // hands they raised preflop
	ThreeBetSpots int // hands they acted preflop facing one raise
	ThreeBets     int // and reraised
	Aggressive    int // bets and raises
	Calls         int
	FacedBets     int // times they acted facing a bet after the flop
	FoldedToBets  int // and folded
}

// Decide picks a move for the bot called name in seat, whose turn it is. view
// should be that player's own view, so the bot only knows what they would.
// line is every action this hand so far, blinds first, and reads are what's
// known of the other players by seat. Both are public.
func Decide(level, name string, view holdem.View, seat int, line []holdem.Acted, reads map[int]Read, r *rand.Rand) holdem.Action {
	s := spotOf(view, seat, line)
	s.style = styleOf(name)
	s.reads = reads
	return toHoldem(s, choose(level, s, r))
}

func choose(level string, s spot, r *rand.Rand) Action {
	switch level {
	case Medium:
		return medium(s, r)
	case Hard:
		return hard(s, r)
	}
	return easy(s, r)
}

func toHoldem(s spot, a Action) holdem.Action {
	switch a.Kind {
	case Raise:
		return holdem.Raise{To: s.bet + a.Amount}
	case Call:
		return holdem.Call{}
	case Check:
		return holdem.Check{}
	}
	return holdem.Fold{}
}

// spot is what a bot looks at to decide.
type spot struct {
	game     holdem.Game
	hole     []eval.Card
	board    []eval.Card
	top      uint // this street's biggest bet
	bet      uint // what the bot has bet this street
	stack    uint
	minRaise uint
	called   bool // already acted, so it can't raise unless someone raised
	pot      uint // every chip bet this hand, this street included
	bb       uint // the big blind, or the last straddle
	others   int  // opponents still in the hand
	behind   int  // opponents who act after the bot after the flop, 0 on the button
	sb, bbs  bool // the bot posted the small blind, or the big blind or last straddle
	limpers  int  // opponents who just called the big blind preflop
	led      bool // the bot made the last bet or raise before this street
	raised   bool // someone bet or raised before this street
	style    style

	view  holdem.View
	seat  int
	line  []holdem.Acted
	reads map[int]Read
}

// aggressor is the seat that made the last bet or raise before this street, 0
// if nobody did.
func aggressor(line []holdem.Acted, street holdem.Street) int {
	seat := 0
	for _, a := range line {
		if a.Street == street {
			break
		}
		if a.Kind == holdem.KindBet || a.Kind == holdem.KindRaise {
			seat = a.Seat
		}
	}
	return seat
}

func spotOf(view holdem.View, seat int, line []holdem.Acted) spot {
	aggressor := aggressor(line, view.Street)
	s := spot{
		view:     view,
		seat:     seat,
		line:     line,
		game:     view.Config.Game,
		board:    view.Board,
		bb:       view.Config.BigBlind << len(view.Straddles),
		minRaise: view.MinRaise,
		led:      aggressor == seat,
		raised:   aggressor != 0,
	}
	big := view.BB
	if n := len(view.Straddles); n > 0 {
		big = view.Straddles[n-1]
	}
	s.sb, s.bbs = seat == view.SB, seat == big
	me := 0
	for i, p := range view.Players {
		s.top = max(s.top, p.Bet)
		s.pot += p.Committed
		switch {
		case p.Seat == seat:
			me = i
			s.hole, s.bet, s.stack, s.called = p.Cards, p.Bet, p.Stack, p.Acted
		case p.In:
			s.others++
		}
	}
	// after the flop the seat left of the button acts first, the button last
	n := len(view.Players)
	for k := 1; k < n && seat != view.Button; k++ {
		p := view.Players[(me+k)%n]
		if p.In {
			s.behind++
		}
		if p.Seat == view.Button {
			break
		}
	}
	if s.top == s.bb {
		for _, p := range view.Players {
			if p.In && p.Bet == s.bb && p.Seat != big && p.Seat != seat {
				s.limpers++
			}
		}
	}
	return s
}

// easy is a calling station: it checks when it can, calls most bets, folds
// weak hands to big bets, and sometimes min-raises a strong hand.
func easy(s spot, r *rand.Rand) Action {
	owe := s.top - s.bet
	h := strengthOf(s.game, s.hole, s.board)

	if h == strong && !s.called && r.Float64() < 0.3 {
		raise := s.top + s.minRaise - s.bet
		if raise < s.stack {
			return Action{Kind: Raise, Amount: raise}
		}
	}
	if owe == 0 {
		return Action{Kind: Check}
	}
	big := owe*3 > s.stack
	switch {
	case h == weak && big,
		h == weak && r.Float64() < 0.15,
		h == fair && big && r.Float64() < 0.5:
		return Action{Kind: Fold}
	}
	return Action{Kind: Call}
}

type strength int

const (
	weak strength = iota
	fair
	strong
)

// strengthOf roughly rates a hand: preflop by its two cards, after the flop
// by the best hand it makes with the board.
func strengthOf(g holdem.Game, hole, board []eval.Card) strength {
	if len(hole) < 2 || hole[0] == 0 || hole[1] == 0 {
		return weak
	}
	if len(board) < 3 {
		return preflop(hole)
	}
	// lower scores are better hands, see holdem.HandName
	switch _, score := g.Best(hole, board); {
	case score <= 3325: // two pair or better
		return strong
	case score <= 6185: // a pair
		return fair
	}
	return weak
}

func preflop(hole []eval.Card) strength {
	if len(hole) > 2 {
		// Omaha: as good as the best two cards
		best := weak
		for i := range hole {
			for j := i + 1; j < len(hole); j++ {
				best = max(best, preflop([]eval.Card{hole[i], hole[j]}))
			}
		}
		return best
	}
	a, b := rank(hole[0]), rank(hole[1])
	suited := hole[0]&hole[1]&0xF000 != 0
	switch {
	case a == b && a >= 8: // tens or better
		return strong
	case a == b, min(a, b) >= 9, max(a, b) == 12 && suited: // any pair, two jacks or higher, suited ace
		return fair
	}
	return weak
}

// rank is a card's rank, from 0 for a deuce to 12 for an ace.
func rank(c eval.Card) int {
	return int(c>>8) & 0xF
}
