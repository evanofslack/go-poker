package bot

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand/v2"

	"github.com/alexclewontin/riverboat/eval"
)

// trials is how many runouts medium simulates to estimate its equity.
const trials = 1000

// medium plays a tight, aggressive game. Preflop it opens by position and
// reraises strong hands. After the flop it weighs its equity against the
// hands opponents would still hold, keeps betting hands it raised, bets and
// raises draws as semi-bluffs, and sizes bets by the board, not its hand.
func medium(s spot, r *rand.Rand) Action {
	if len(s.board) < 3 {
		return mediumPreflop(s, r)
	}
	owe := s.top - s.bet
	river := len(s.board) == 5
	d := drawOf(s.hole, s.board)
	w := wet(s.board)

	// opponents still in played their hands, and tighter ones in a raised pot
	// or when they bet
	minChen := 3.0
	if s.raised {
		minChen = 6
	}
	e := equity(s.hole, s.board, s.others, minChen, owe > 0, trials, r)
	why := fmt.Sprintf("equity %.2f, %s", e, d)

	if owe == 0 {
		value := 0.5 + 0.07*float64(s.others)
		cbet := 0.65
		if s.others > 1 {
			cbet = 0.35
		}
		switch {
		case e > 0.85 && !w && r.Float64() < 0.15:
			return because(Action{Kind: Check}, "%s, slow-play", why)
		case e > value:
			return because(s.size(betSize(s, w, r)), "%s > %.2f, value bet", why, value)
		case s.led && len(s.board) == 3 && s.often(cbet, r):
			return because(s.size(betSize(s, w, r)), "%s, c-bet", why)
		case !river && d >= openEnded && s.often(0.5, r),
			!river && d == gutshot && s.often(0.2, r):
			return because(s.size(betSize(s, w, r)), "%s, semi-bluff", why)
		case s.led && len(s.board) == 4 && e > 0.4 && s.often(0.5, r):
			return because(s.size(betSize(s, w, r)), "%s, barrel", why)
		case river && s.led && e < 0.3 && s.often(0.25, r):
			return because(s.size(betSize(s, w, r)), "%s, river bluff", why)
		case s.others == 1 && s.behind == 0 && s.often(0.25, r):
			return because(s.size(betSize(s, w, r)), "%s, stab when checked to", why)
		}
		return because(Action{Kind: Check}, "%s, check", why)
	}

	odds := float64(owe) / float64(s.pot+owe)
	credit := 0.0 // for the chips a draw wins when it hits
	switch {
	case river:
	case d >= openEnded:
		credit = 0.08
	case d == gutshot:
		credit = 0.03
	}
	switch {
	case e > 0.7:
		return because(s.size(raiseSize(s, owe, w, r)), "%s > 0.7, raise for value", why)
	case !river && d >= openEnded && s.often(0.15, r):
		return because(s.size(raiseSize(s, owe, w, r)), "%s, semi-bluff raise", why)
	case e+credit > odds:
		return because(Action{Kind: Call}, "%s + %.2f > odds %.2f, call", why, credit, odds)
	}
	return because(Action{Kind: Fold}, "%s + %.2f <= odds %.2f, fold", why, credit, odds)
}

// betSize is a bet for value or as a bluff alike, so it doesn't show the hand:
// a third to half the pot on a dry board, a half to most of it on a wet one,
// and more again on the river.
func betSize(s spot, wet bool, r *rand.Rand) uint {
	lo, hi := 0.33, 0.5
	switch {
	case len(s.board) == 5:
		lo, hi = 0.6, 1
	case wet:
		lo, hi = 0.5, 0.85
	}
	return uint((lo + (hi-lo)*r.Float64()) * float64(s.pot))
}

// raiseSize calls owe, then raises by a bet's worth of the pot after calling.
func raiseSize(s spot, owe uint, wet bool, r *rand.Rand) uint {
	after := s
	after.pot += owe
	return owe + betSize(after, wet, r)
}

// draw is the best draw a hand has to a flush or straight.
type draw int

const (
	noDraw draw = iota
	gutshot
	openEnded // or a flush draw
	comboDraw // a flush draw and a straight draw
)

func (d draw) String() string {
	return [...]string{"no draw", "gutshot", "draw", "combo draw"}[d]
}

// drawOf is the draw hole has on board, counting only draws that use a hole
// card and leaving out hands already made.
func drawOf(hole [2]eval.Card, board []eval.Card) draw {
	if len(board) >= 5 {
		return noDraw
	}
	all := append([]eval.Card{hole[0], hole[1]}, board...)
	flush := false
	for _, suit := range []eval.Card{0x1000, 0x2000, 0x4000, 0x8000} {
		n := 0
		for _, c := range all {
			if c&suit != 0 {
				n++
			}
		}
		if n == 4 && (hole[0]&suit != 0 || hole[1]&suit != 0) {
			flush = true
		}
	}
	mine, theirs := rankMask(all), rankMask(board)
	outs := 0
	if !straight(mine) {
		for x := range 13 {
			add := uint16(1) << x
			if mine&add == 0 && straight(mine|add) && !straight(theirs|add) {
				outs++
			}
		}
	}
	switch {
	case flush && outs > 0:
		return comboDraw
	case flush || outs >= 2:
		return openEnded
	case outs == 1:
		return gutshot
	}
	return noDraw
}

// wet is a board with two or more of a suit, or three cards within a
// straight's reach, where hands and draws change a lot from street to street.
func wet(board []eval.Card) bool {
	for _, suit := range []eval.Card{0x1000, 0x2000, 0x4000, 0x8000} {
		n := 0
		for _, c := range board {
			if c&suit != 0 {
				n++
			}
		}
		if n >= 2 && len(board) == 3 || n >= 3 {
			return true
		}
	}
	m := rankMask(board)
	for lo := -1; lo <= 8; lo++ {
		if bits.OnesCount16(window(m, lo)) >= 3 {
			return true
		}
	}
	return false
}

// rankMask has bit r set for each rank r in cards, 0 for a deuce.
func rankMask(cards []eval.Card) uint16 {
	var m uint16
	for _, c := range cards {
		m |= 1 << rank(c)
	}
	return m
}

// window is the ranks in m from lo to lo+4, where -1 is an ace played low.
func window(m uint16, lo int) uint16 {
	if lo < 0 {
		return (m&0xF)<<1 | m>>12&1 // ace, deuce to five
	}
	return m >> lo & 0x1F
}

// straight reports whether ranks m make a straight.
func straight(m uint16) bool {
	for lo := -1; lo <= 8; lo++ {
		if window(m, lo) == 0x1F {
			return true
		}
	}
	return false
}

// mediumPreflop raises or folds when nobody has raised, with more hands the
// later it acts, and reraises strong hands and a few bluffs.
func mediumPreflop(s spot, r *rand.Rand) Action {
	c := chen(s.hole) + s.style.loose
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
		bluff := suitedPlayable(s.hole) && s.often(0.25, r)
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

// anyHand is a chen score below every hand's, for equity against random hands.
const anyHand = -2

// equity estimates the chance hole wins against others hands scoring at
// least minChen, by dealing out the rest of the board trials times. Against
// a bet, opponents mostly hold a pair or a draw. A split counts as a share.
func equity(hole [2]eval.Card, board []eval.Card, others int, minChen float64, bet bool, trials int, r *rand.Rand) float64 {
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
		// swap hands an opponent wouldn't play for cards past the runout
		for o := range others {
			for try := 0; try < 10 && !plays([2]eval.Card{deck[2*o], deck[2*o+1]}, board, minChen, bet, r); try++ {
				for _, k := range []int{2 * o, 2*o + 1} {
					j := need + r.IntN(len(deck)-need)
					deck[k], deck[j] = deck[j], deck[k]
				}
			}
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

// plays reports whether an opponent would still be in with hole: a hand
// scoring minChen preflop, and against a bet one that connects with the board,
// or now and then a bluff.
func plays(hole [2]eval.Card, board []eval.Card, minChen float64, bet bool, r *rand.Rand) bool {
	if chen(hole) < minChen {
		return false
	}
	if !bet || len(board) < 3 || r.Float64() < 0.15 {
		return true
	}
	return connects(hole, board)
}

// connects reports whether hole makes a pair or better with board, using a
// hole card, or has a draw.
func connects(hole [2]eval.Card, board []eval.Card) bool {
	b := rankMask(board)
	if rank(hole[0]) == rank(hole[1]) || b&rankMask(hole[:]) != 0 {
		return true
	}
	if straight(rankMask(append([]eval.Card{hole[0], hole[1]}, board...))) && !straight(b) {
		return true
	}
	return drawOf(hole, board) != noDraw
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
