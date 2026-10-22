package bot

import (
	"fmt"
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
		case e > 0.6 && !slowPlay:
			return because(s.size(s.pot*2/3), "equity %.2f > 0.6, bet 2/3 pot", e)
		case e < 0.3 && r.Float64() < 0.1:
			return because(s.size(s.pot*2/3), "equity %.2f, bluff 2/3 pot", e)
		case slowPlay:
			return because(Action{Kind: Check}, "equity %.2f, slow-play", e)
		}
		return because(Action{Kind: Check}, "equity %.2f <= 0.6", e)
	}
	odds := float64(owe) / float64(s.pot+owe)
	switch {
	case e > 0.75 && !slowPlay:
		// a pot-sized raise: call, then raise by the pot after calling
		return because(s.size(owe+s.pot+owe), "equity %.2f > 0.75, raise pot", e)
	case e <= odds && r.Float64() < 0.03:
		return because(s.size(owe+s.pot+owe), "equity %.2f <= odds %.2f, bluff raise pot", e, odds)
	case e > odds+0.05:
		return because(Action{Kind: Call}, "equity %.2f > odds %.2f + 0.05", e, odds)
	}
	return because(Action{Kind: Fold}, "equity %.2f <= odds %.2f + 0.05", e, odds)
}

// mediumPreflop raises or folds when nobody has raised, with more hands the
// later it acts, and reraises strong hands and a few bluffs.
func mediumPreflop(s spot, r *rand.Rand) Action {
	c := chen(s.hole)
	owe := s.top - s.bet
	switch {
	case s.top <= s.bb: // unopened, maybe limped
		if c >= openAt(s) {
			// open to two and a half or three blinds, plus one for each limper
			to := s.bb*5/2 + s.bb*uint(r.IntN(2))/2 + s.bb*uint(s.limpers)
			return because(s.size(jitter(to, s.bb, r)-s.bet), "chen %v >= %v, open", c, openAt(s))
		}
		if owe == 0 {
			return because(Action{Kind: Check}, "chen %v, free check", c)
		}
		return because(Action{Kind: Fold}, "chen %v < %v, fold unopened", c, openAt(s))
	case s.top <= 6*s.bb: // one raise
		bluff := suitedPlayable(s.hole) && r.Float64() < 0.25
		if c >= 10 || s.others == 1 && c >= 8 || bluff {
			// reraise to three times the raise in position, four out of it
			mult := uint(3)
			if s.behind > 0 {
				mult = 4
			}
			return because(s.size(jitter(mult*s.top, s.bb, r)-s.bet), "chen %v, 3-bet (bluff %v)", c, bluff)
		}
		need := 8.0
		switch {
		case owe*3 > s.stack:
			need = 10
		case s.others == 1:
			need = 5 // heads up, defend wide
		case s.bbs:
			need = 7 // the big blind is already part way in
		case s.behind > 0:
			need = 9
		}
		if c >= need {
			return because(Action{Kind: Call}, "chen %v >= %v, call a raise", c, need)
		}
		return because(Action{Kind: Fold}, "chen %v < %v, fold to a raise", c, need)
	}
	// a reraise or more
	switch {
	case c >= 13:
		return because(s.size(jitter(s.top*9/4, s.bb, r)-s.bet), "chen %v >= 13, 4-bet", c)
	case c >= 11:
		return because(Action{Kind: Call}, "chen %v >= 11, call a 3-bet", c)
	}
	return because(Action{Kind: Fold}, "chen %v < 11, fold to a 3-bet", c)
}

// openAt is the chen score medium opens with: more hands closer to the
// button, where fewer players are left to act after it.
func openAt(s spot) float64 {
	switch {
	case s.others == 1:
		return 3 // heads up, most hands
	case s.sb:
		return 5
	case s.bbs:
		return 9 // only raises limpers with good hands
	}
	return float64(4 + min(s.behind, 3))
}

// suitedPlayable is a suited ace or suited connector, a hand worth reraising
// as a bluff now and then.
func suitedPlayable(hole [2]eval.Card) bool {
	if hole[0]&hole[1]&0xF000 == 0 {
		return false
	}
	a, b := rank(hole[0]), rank(hole[1])
	return max(a, b) == 12 || (a-b == 1 || b-a == 1) && min(a, b) >= 3
}

// jitter varies chips by up to a tenth either way, in whole big blinds, so
// sizes don't give the hand away.
func jitter(chips, bb uint, r *rand.Rand) uint {
	c := float64(chips) * (0.9 + 0.2*r.Float64())
	return max(uint(c/float64(bb)+0.5), 1) * bb
}

// because adds why a bot chose a, before any reason a already has.
func because(a Action, format string, args ...any) Action {
	reason := fmt.Sprintf(format, args...)
	if a.Reason != "" {
		reason += ", " + a.Reason
	}
	a.Reason = reason
	return a
}

// size turns a bet or raise of about add chips into a legal move: at least a
// min-raise, and all in once that's the whole stack. A bot that can't raise
// checks or calls instead.
func (s spot) size(add uint) Action {
	owe := s.top - s.bet
	switch {
	case s.called && owe == 0:
		return Action{Kind: Check, Reason: "can't raise, check"}
	case s.called, s.stack <= owe:
		return Action{Kind: Call, Reason: "can't raise, call"}
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
