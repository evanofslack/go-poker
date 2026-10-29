package bot

import (
	"math"
	"math/rand/v2"
	"slices"
	"sort"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// Hard reads each opponent's range as weights over a pool of hands, from
// what they've done this hand, and deals them hands by those weights to work
// out its equity.

// omahaPool is how many Omaha hands stand in for every hand an opponent
// could hold.
const omahaPool = 600

// pool is the hands an opponent might hold: every two card hand that doesn't
// use a card the bot knows, or a sample of Omaha hands. How strong a hand is
// on a board is its percentile among the pool, so it reads the same in every
// game.
type pool struct {
	g      holdem.Game
	hands  [][]eval.Card
	masks  []uint64
	boards [][]eval.Card // the boards so far, two on double board
	deck   []eval.Card   // the cards the bot doesn't know
	known  uint64
	pre    []float64         // starting hand percentiles
	made   map[int][]float64 // by board length: made hand percentile over the boards
	draws  map[int][]draw    // by board length: the best draw on any board
}

func newPool(g holdem.Game, hole []eval.Card, boards [][]eval.Card, r *rand.Rand) *pool {
	p := &pool{g: g, boards: boards, made: map[int][]float64{}, draws: map[int][]draw{}}
	p.known = maskOf(hole)
	for _, b := range boards {
		p.known |= maskOf(b)
	}
	for _, c := range holdem.Deck(g) {
		if p.known&cardBit(c) == 0 {
			p.deck = append(p.deck, c)
		}
	}
	add := func(h []eval.Card) {
		p.hands = append(p.hands, h)
		p.masks = append(p.masks, maskOf(h))
		p.pre = append(p.pre, percentile(g, h))
	}
	if n := g.HoleCards(); n > 2 {
		deck := slices.Clone(p.deck)
		for range omahaPool {
			for i := range n {
				j := i + r.IntN(len(deck)-i)
				deck[i], deck[j] = deck[j], deck[i]
			}
			add(slices.Clone(deck[:n]))
		}
	} else {
		for i, a := range p.deck {
			for _, b := range p.deck[i+1:] {
				add([]eval.Card{a, b})
			}
		}
	}
	return p
}

// strength is each hand's made percentile and best draw on the first n cards
// of the boards. On double board it's the percentile of a hand's average
// over the two, since each board is half the pot.
func (p *pool) strength(n int) ([]float64, []draw) {
	if made, ok := p.made[n]; ok {
		return made, p.draws[n]
	}
	made := make([]float64, len(p.hands))
	draws := make([]draw, len(p.hands))
	scores := make([]int, len(p.hands))
	for _, b := range p.boards {
		for i, h := range p.hands {
			_, scores[i] = p.g.Best(h, b[:n])
			draws[i] = max(draws[i], drawOf(h, b[:n]))
		}
		sorted := slices.Sorted(slices.Values(scores))
		for i, s := range scores {
			// lower scores are better: the share of hands that score worse,
			// half the ties
			worse := len(sorted) - sort.SearchInts(sorted, s+1)
			ties := sort.SearchInts(sorted, s+1) - sort.SearchInts(sorted, s) - 1
			made[i] += (float64(worse) + float64(ties)/2) / float64(len(sorted)-1)
		}
	}
	if len(p.boards) > 1 {
		made = rerank(made)
	}
	p.made[n], p.draws[n] = made, draws
	return made, draws
}

// rerank turns xs into percentiles among themselves.
func rerank(xs []float64) []float64 {
	sorted := slices.Sorted(slices.Values(xs))
	out := make([]float64, len(xs))
	for i, x := range xs {
		lo := sort.SearchFloat64s(sorted, x)
		hi := sort.Search(len(sorted), func(k int) bool { return sorted[k] > x })
		out[i] = (float64(lo) + float64(hi-lo-1)/2) / float64(len(sorted)-1)
	}
	return out
}

// odds deals the hand out trials times, each opponent's hand drawn by its
// weights, and returns hole's share of the pot. It also returns that share
// with each deal weighted by how likely every opponent is to continue, for
// each set of conts (by opponent, by hand).
func (p *pool) odds(hole []eval.Card, ws [][]float64, conts [][][]float64, trials int, r *rand.Rand) (float64, []float64) {
	cums := make([][]float64, len(ws))
	for o, w := range ws {
		cums[o] = make([]float64, len(w))
		sum := 0.0
		for i, x := range w {
			sum += x
			cums[o][i] = sum
		}
	}
	full := make([]eval.Card, 5)
	idx := make([]int, len(ws))
	var total, n float64
	num, den := make([]float64, len(conts)), make([]float64, len(conts))
deal:
	for range trials {
		used := p.known
		for o, cum := range cums {
			idx[o] = -1
			for range 40 {
				i := sort.SearchFloat64s(cum, r.Float64()*cum[len(cum)-1])
				if i < len(cum) && p.masks[i]&used == 0 {
					idx[o] = i
					used |= p.masks[i]
					break
				}
			}
			if idx[o] < 0 {
				continue deal
			}
		}
		share := 0.0
		for _, b := range p.boards {
			copy(full, b)
			for k := len(b); k < 5; k++ {
				for {
					c := p.deck[r.IntN(len(p.deck))]
					if used&cardBit(c) == 0 {
						full[k] = c
						used |= cardBit(c)
						break
					}
				}
			}
			_, mine := p.g.Best(hole, full)
			ties := 1
			for _, i := range idx {
				_, theirs := p.g.Best(p.hands[i], full)
				if theirs < mine {
					ties = 0
					break
				}
				if theirs == mine {
					ties++
				}
			}
			if ties > 0 {
				share += 1 / float64(ties) / float64(len(p.boards))
			}
		}
		total += share
		n++
		for j, cont := range conts {
			w := 1.0
			for o, i := range idx {
				w *= cont[o][i]
			}
			num[j] += w * share
			den[j] += w
		}
	}
	for j := range num {
		num[j] /= max(den[j], 1e-9)
	}
	return total / max(n, 1), num
}

// tendencies is how an opponent plays, from their reads, shrunk toward a
// typical player's until they've played enough hands to tell.
type tendencies struct {
	vpip, pfr, threeBet float64 // shares of hands
	aggro               float64 // bets and raises per call, over a typical player's
	fold                float64 // share of bets they fold to after the flop
}

func tendenciesOf(rd Read) tendencies {
	const k = 20 // hands of a typical player to start from
	share := func(n, of int, typical float64) float64 {
		return (float64(n) + typical*k) / (float64(of) + k)
	}
	return tendencies{
		vpip:     share(rd.VPIP, rd.Hands-rd.Bombs, 0.25),
		pfr:      share(rd.PFR, rd.Hands-rd.Bombs, 0.18),
		fold:     share(rd.FoldedToBets, rd.FacedBets, 0.45),
		threeBet: share(rd.ThreeBets, rd.ThreeBetSpots, 0.07),
		aggro:    (float64(rd.Aggressive) + 1.5*k) / (float64(rd.Calls) + k) / 1.5,
	}
}

// soft is near 1 for a hand in the top width of hands by percentile p and
// near floor below, with a short slope between.
func soft(p, width, floor float64) float64 {
	return floor + (1-floor)/(1+math.Exp(-(p-(1-width))/0.025))
}

// openWidth is the share of hands a player opens with behind players left to
// act preflop, of dealt in all.
func openWidth(behind, dealt int) float64 {
	if dealt == 2 {
		return 0.8
	}
	switch behind {
	case 0, 1:
		return 0.45 // the small blind, or the big blind raising limpers
	case 2:
		return 0.42 // the button
	case 3:
		return 0.28
	case 4:
		return 0.2
	}
	return 0.15
}

// wider is how much wider than hold'em players play g preflop.
func wider(g holdem.Game) float64 {
	if g.HoleCards() > 2 {
		return 1.5
	}
	return 1
}

// preWeight weighs a hand at percentile p by a preflop action of kind, made
// facing raises raises with behind players left to act.
func preWeight(kind string, raises, behind, dealt int, t tendencies, g holdem.Game) func(p float64) float64 {
	widen := func(w, seen, typical float64) float64 {
		return min(w*min(max(seen/typical, 0.35), 3)*wider(g), 0.95)
	}
	open := widen(openWidth(behind, dealt), t.pfr, 0.18)
	switch {
	case kind == holdem.KindCheck:
		// the big blind would raise its best hands
		return func(p float64) float64 { return 1 - 0.6*soft(p, 0.08, 0) }
	case kind == holdem.KindRaise && raises == 0:
		return func(p float64) float64 { return soft(p, open, 0.01) }
	case kind == holdem.KindRaise && raises == 1:
		w := widen(0.07, t.threeBet, 0.07)
		return func(p float64) float64 { return soft(p, w, 0.005) }
	case kind == holdem.KindRaise:
		return func(p float64) float64 { return soft(p, 0.03, 0.002) }
	case raises == 0: // a limp
		w := widen(0.45, t.vpip, 0.25)
		return func(p float64) float64 { return soft(p, w, 0.02) * (1 - 0.6*soft(p, open/3, 0)) }
	case raises == 1:
		w := 0.18
		if behind == 0 || dealt == 2 {
			w = 0.4 // the big blind, priced in
		}
		w = widen(w, t.vpip, 0.25)
		return func(p float64) float64 { return soft(p, w, 0.01) * (1 - 0.6*soft(p, 0.03, 0)) }
	case raises == 2:
		return func(p float64) float64 { return soft(p, 0.12, 0.005) * (1 - 0.6*soft(p, 0.025, 0)) }
	}
	return func(p float64) float64 { return soft(p, 0.05, 0.002) }
}

// postWeight weighs a hand by an action of kind after the flop, from its
// made percentile on the board then and its draw. x is the chips the action
// put in over the pot before it.
func postWeight(kind string, made float64, d draw, x float64, river bool, t tendencies) float64 {
	switch kind {
	case holdem.KindBet, holdem.KindRaise:
		// bigger bets are more polar: strong hands and air. Passive players
		// bet fewer weaker hands, aggressive ones more.
		value := 0.8 + 0.04*min(x, 2)
		loose := min(max(t.aggro, 0.3), 2)
		mid := max(0.05, 0.55-0.35*x) * loose
		if kind == holdem.KindRaise {
			value += 0.06
			mid /= 2
		}
		switch {
		case made >= value:
			return 1
		case !river && d >= openEnded:
			return 0.6
		case !river && d == gutshot:
			return 0.3
		case made < 0.4:
			return 0.15 * loose
		}
		return min(mid, 1)
	case holdem.KindCall:
		switch {
		case made >= 0.93:
			return 0.55 // would often raise
		case made >= 0.55:
			return 1
		case !river && d >= gutshot:
			return 0.9
		case made >= 0.35:
			return max(0.1, 0.7-0.35*x)
		}
		return 0.1
	case holdem.KindCheck:
		if made >= 0.88 {
			return 0.45
		}
	}
	return 1
}

// weigh is each opponent still in's weights over the pool, by seat, from
// every action they've made this hand.
func weigh(s spot, p *pool) map[int][]float64 {
	ws := map[int][]float64{}
	dealt := 0
	for _, pl := range s.view.Players {
		if pl.Dealt {
			dealt++
		}
		if pl.In && pl.Seat != s.seat {
			w := make([]float64, len(p.hands))
			for i := range w {
				w[i] = 1
			}
			ws[pl.Seat] = w
		}
	}
	var street holdem.Street
	var pot uint
	raises, order := 0, 0
	position := map[int]int{} // preflop, how many players acted first
	for _, a := range s.line {
		if a.Street != street {
			street = a.Street
		}
		before := pot
		pot += a.Amount
		switch a.Kind {
		case holdem.KindAnte, holdem.KindBomb, holdem.KindSmallBlind, holdem.KindBigBlind, holdem.KindStraddle, holdem.KindFold:
			continue
		}
		if street == holdem.Preflop {
			pos, ok := position[a.Seat]
			if !ok {
				pos = order
				position[a.Seat] = pos
				order++
			}
			if w, ok := ws[a.Seat]; ok {
				f := preWeight(a.Kind, raises, dealt-pos-1, dealt, tendenciesOf(s.reads[a.Seat]), p.g)
				for i := range w {
					w[i] *= f(p.pre[i])
				}
			}
			if a.Kind == holdem.KindRaise {
				raises++
			}
			continue
		}
		w, ok := ws[a.Seat]
		if !ok {
			continue
		}
		n := map[holdem.Street]int{holdem.Flop: 3, holdem.Turn: 4, holdem.River: 5}[street]
		if n == 0 || n > len(s.board) {
			continue
		}
		made, draws := p.strength(n)
		x := float64(a.Amount) / float64(max(before, 1))
		t := tendenciesOf(s.reads[a.Seat])
		for i := range w {
			w[i] *= postWeight(a.Kind, made[i], draws[i], x, n == 5, t)
		}
	}
	return ws
}
