package bot

import (
	"math"
	"math/rand/v2"

	"github.com/alexclewontin/riverboat/eval"
)

// trials is how many runouts medium simulates to estimate its equity.
const trials = 1000

// medium plays by the numbers: a starting hand score preflop, then its equity
// against random hands versus the price of calling. It bets for value, bluffs
// now and then, and sometimes slow-plays a monster.
func medium(s spot, r *rand.Rand) Action {
	if len(s.board) < 3 {
		return mediumPreflop(s, r)
	}
	owe := s.top - s.bet
	e := equity(s.hole, s.board, s.others, trials, r)
	slowPlay := e > 0.85 && r.Float64() < 0.2

	if owe == 0 {
		switch {
		case e > 0.6 && !slowPlay, e < 0.3 && r.Float64() < 0.1:
			return s.size(s.pot * 2 / 3)
		}
		return Action{Kind: Check}
	}
	odds := float64(owe) / float64(s.pot+owe)
	switch {
	case e > 0.75 && !slowPlay, e <= odds && r.Float64() < 0.03:
		// a pot-sized raise: call, then raise by the pot after calling
		return s.size(owe + s.pot + owe)
	case e > odds+0.05:
		return Action{Kind: Call}
	}
	return Action{Kind: Fold}
}

func mediumPreflop(s spot, r *rand.Rand) Action {
	c := chen(s.hole)
	if s.others == 1 {
		c += 2 // heads up, more hands are worth playing
	}
	owe := s.top - s.bet
	big := owe*3 > s.stack
	switch {
	case c >= 10 && r.Float64() < 0.8:
		return s.size(3*s.top - s.bet) // raise to three times the bet
	case owe == 0:
		return Action{Kind: Check}
	case big && c >= 9,
		!big && c >= 7,
		s.top <= s.bb && c >= 5: // limp along when nobody raised
		return Action{Kind: Call}
	}
	return Action{Kind: Fold}
}

// size turns a bet or raise of about add chips into a legal move: at least a
// min-raise, and all in once that's the whole stack. A bot that can't raise
// checks or calls instead.
func (s spot) size(add uint) Action {
	owe := s.top - s.bet
	switch {
	case s.called && owe == 0:
		return Action{Kind: Check}
	case s.called, s.stack <= owe:
		return Action{Kind: Call}
	}
	add = max(add, owe+s.minRaise)
	if add >= s.stack {
		return Action{Kind: Raise, Amount: s.stack}
	}
	return Action{Kind: Raise, Amount: add}
}

// equity estimates the chance hole wins against others random hands, by
// dealing out the rest of the board trials times. A split counts as a share.
func equity(hole [2]eval.Card, board []eval.Card, others, trials int, r *rand.Rand) float64 {
	if others == 0 {
		return 1
	}
	known := map[eval.Card]bool{hole[0]: true, hole[1]: true}
	for _, c := range board {
		known[c] = true
	}
	var deck []eval.Card
	for _, c := range eval.DefaultDeck {
		if !known[c] {
			deck = append(deck, c)
		}
	}
	need := 2*others + 5 - len(board)
	full := make([]eval.Card, 5)
	copy(full, board)

	var won float64
	for range trials {
		// shuffle just the cards this runout needs to the front
		for i := range need {
			j := i + r.IntN(len(deck)-i)
			deck[i], deck[j] = deck[j], deck[i]
		}
		copy(full[len(board):], deck[2*others:need])
		mine := score(hole[0], hole[1], full)
		ties := 1
		for o := range others {
			theirs := score(deck[2*o], deck[2*o+1], full)
			if theirs < mine {
				ties = 0
				break
			}
			if theirs == mine {
				ties++
			}
		}
		if ties > 0 {
			won += 1 / float64(ties)
		}
	}
	return won / float64(trials)
}

// score is the best hand two hole cards make with a full board, lower is better.
func score(a, b eval.Card, board []eval.Card) int {
	_, s := eval.BestFiveOfSeven(a, b, board[0], board[1], board[2], board[3], board[4])
	return s
}

// chen scores a starting hand with Bill Chen's formula, from -1 for 7-2 off
// suit up to 20 for aces.
func chen(hole [2]eval.Card) float64 {
	hi, lo := rank(hole[0]), rank(hole[1])
	if lo > hi {
		hi, lo = lo, hi
	}
	value := func(rank int) float64 {
		switch rank {
		case 12:
			return 10
		case 11:
			return 8
		case 10:
			return 7
		case 9:
			return 6
		}
		return float64(rank+2) / 2
	}
	if hi == lo {
		return math.Max(value(hi)*2, 5)
	}
	c := value(hi)
	if hole[0]&hole[1]&0xF000 != 0 {
		c += 2
	}
	switch gap := hi - lo - 1; {
	case gap == 1:
		c--
	case gap == 2:
		c -= 2
	case gap == 3:
		c -= 4
	case gap >= 4:
		c -= 5
	}
	// small connected cards can make straights
	if hi-lo <= 2 && hi < 10 {
		c++
	}
	return math.Ceil(c)
}
