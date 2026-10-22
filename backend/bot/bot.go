// Package bot picks moves for computer players. It only reads a game view, so
// it can't change the game or see more than the view shows.
package bot

import (
	"math/rand/v2"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/poker"
)

// bot levels
const (
	Easy   = "easy"
	Medium = "medium"
)

// Kind is the kind of move a bot makes.
type Kind int

const (
	Check Kind = iota
	Call
	Raise
	Fold
)

// Action is a bot's move. Amount is the chips it puts in, for Raise only.
// Reason says why the bot chose it, for logs.
type Action struct {
	Kind   Kind
	Amount uint
	Reason string
}

// Valid reports whether level is a bot level.
func Valid(level string) bool {
	return level == Easy || level == Medium
}

// Decide picks a move for player pn, whose turn it is. view should be pn's own
// player view, so the bot only knows what pn would. aggressor is the player
// who made the last bet or raise before this street, -1 if nobody did.
func Decide(level string, view *poker.GameView, pn uint, aggressor int, r *rand.Rand) Action {
	s := spotOf(view, pn)
	s.led, s.raised = aggressor == int(pn), aggressor >= 0
	if level == Medium {
		return medium(s, r)
	}
	return easy(s, r)
}

// spot is what a bot looks at to decide.
type spot struct {
	hole     [2]eval.Card
	board    []eval.Card
	top      uint // this street's biggest bet
	bet      uint // what the bot has bet this street
	stack    uint
	minRaise uint
	called   bool // already acted, so it can't raise unless someone raised
	pot      uint // every chip bet this hand, this street included
	bb       uint // the big blind
	others   int  // opponents still in the hand
	behind   int  // opponents who act after the bot after the flop, 0 on the button
	sb, bbs  bool // the bot is in the small or big blind
	limpers  int  // opponents who just called the big blind preflop
	led      bool // the bot made the last bet or raise before this street
	raised   bool // someone bet or raised before this street
}

func spotOf(view *poker.GameView, pn uint) spot {
	p := view.Players[pn]
	s := spot{
		hole:     p.Cards,
		board:    dealt(view.CommunityCards),
		bet:      p.Bet,
		stack:    p.Stack,
		minRaise: view.MinRaise,
		called:   p.Called,
		bb:       view.Config.BigBlind,
	}
	for i, q := range view.Players {
		s.top = max(s.top, q.Bet)
		s.pot += q.TotalBet
		if q.In && uint(i) != pn {
			s.others++
		}
	}
	// after the flop the seat left of the dealer acts first, the dealer last
	n := uint(len(view.Players))
	for i := (pn + 1) % n; pn != view.DealerNum; i = (i + 1) % n {
		if view.Players[i].In {
			s.behind++
		}
		if i == view.DealerNum {
			break
		}
	}
	s.sb, s.bbs = pn == view.SBNum, pn == view.BBNum
	if s.top == s.bb {
		for i, q := range view.Players {
			if q.In && q.Bet == s.bb && uint(i) != view.BBNum && uint(i) != pn {
				s.limpers++
			}
		}
	}
	return s
}

// dealt is the cards dealt so far. A view always holds five community cards,
// with zeros for the ones still to come.
func dealt(cards []eval.Card) []eval.Card {
	var d []eval.Card
	for _, c := range cards {
		if c != 0 {
			d = append(d, c)
		}
	}
	return d
}

// easy is a calling station: it checks when it can, calls most bets, folds
// weak hands to big bets, and sometimes min-raises a strong hand.
func easy(s spot, r *rand.Rand) Action {
	owe := s.top - s.bet
	h := strengthOf(s.hole, s.board)

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
func strengthOf(hole [2]eval.Card, board []eval.Card) strength {
	if hole[0] == 0 || hole[1] == 0 {
		return weak
	}
	if len(board) < 3 {
		return preflop(hole)
	}
	// lower scores are better hands, see handName in the server
	switch score := best(append([]eval.Card{hole[0], hole[1]}, board...)); {
	case score <= 3325: // two pair or better
		return strong
	case score <= 6185: // a pair
		return fair
	}
	return weak
}

func preflop(hole [2]eval.Card) strength {
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

// best scores the best five card hand in cards, which holds 5 to 7 cards.
func best(cards []eval.Card) int {
	c := cards
	switch len(c) {
	case 5:
		return eval.HandValue(c[0], c[1], c[2], c[3], c[4])
	case 6:
		_, score := eval.BestFiveOfSix(c[0], c[1], c[2], c[3], c[4], c[5])
		return score
	default:
		_, score := eval.BestFiveOfSeven(c[0], c[1], c[2], c[3], c[4], c[5], c[6])
		return score
	}
}
