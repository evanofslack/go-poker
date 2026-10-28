package bot

import (
	"math/rand/v2"

	"github.com/evanofslack/go-poker/holdem"
)

// baselineLevel is medium as it played before its rewrite, for the sims to
// measure against.
const baselineLevel = "baseline"

// decide is choose, plus the baseline level.
func decide(level string, s spot, r *rand.Rand) Action {
	if level == baselineLevel {
		return baseline(s, r)
	}
	return choose(level, s, r)
}
func baseline(s spot, r *rand.Rand) Action {
	if len(s.board) < 3 {
		return baselinePreflop(s, r)
	}
	owe := s.top - s.bet
	e := equity(holdem.Game{}, s.hole, s.board, s.others, anyHand, false, trials, r)
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

func baselinePreflop(s spot, r *rand.Rand) Action {
	c := chen(s.hole)
	if s.others == 1 {
		c += 2 // heads up, more hands are worth playing
	}
	owe := s.top - s.bet
	big := owe*3 > s.stack
	switch {
	case c >= 10 && r.Float64() < 0.8:
		return because(s.size(3*s.top-s.bet), "chen %v >= 10, raise to 3x", c) // raise to three times the bet
	case owe == 0:
		return because(Action{Kind: Check}, "chen %v, free check", c)
	case big && c >= 9:
		return because(Action{Kind: Call}, "chen %v >= 9, call big bet", c)
	case !big && c >= 7:
		return because(Action{Kind: Call}, "chen %v >= 7, call", c)
	case s.top <= s.bb && c >= 5: // limp along when nobody raised
		return because(Action{Kind: Call}, "chen %v >= 5, limp", c)
	}
	return because(Action{Kind: Fold}, "chen %v too weak", c)
}
