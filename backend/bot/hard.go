package bot

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/evanofslack/go-poker/holdem"
)

// hard reads each opponent's range from how they've played the hand and what
// it knows of them, then picks the move worth the most chips. Before the flop
// it plays a chart by position, with push or fold when short. After it, it
// weighs checking, calling and folding against a few bet sizes, by its equity
// against their range and how often they'd fold.
func hard(s spot, r *rand.Rand) Action {
	boards := [][]holdem.Card{s.board}
	if s.view.Double && len(s.view.Boards) > 1 {
		boards = append(boards, s.view.Boards[1])
	}
	p := newPool(s.game, s.hole, boards, r)
	ws := weigh(s, p)
	if len(s.board) < 3 {
		return hardPreflop(s, p, ws, r)
	}
	return hardPostflop(s, p, ws, r)
}

// opponents is the weights of each opponent still in, and their seats, in
// seat order.
func opponents(s spot, ws map[int][]float64, only func(holdem.Player) bool) ([][]float64, []holdem.Player) {
	var w [][]float64
	var ps []holdem.Player
	for _, pl := range s.view.Players {
		if x, ok := ws[pl.Seat]; ok && (only == nil || only(pl)) {
			w = append(w, x)
			ps = append(ps, pl)
		}
	}
	return w, ps
}

// option is a move and what it's worth in chips from here.
type option struct {
	a  Action
	ev float64
}

// pick takes the best option, or now and then one nearly as good, so the
// bot isn't fully predictable. scale is a pot, for what "nearly" means.
func pick(opts []option, scale float64, r *rand.Rand) Action {
	best := opts[0]
	for _, o := range opts[1:] {
		if o.ev > best.ev {
			best = o
		}
	}
	// softmax over options within a percent or two of a pot
	temp := 0.01 * scale
	var weights []float64
	sum := 0.0
	for _, o := range opts {
		w := math.Exp((o.ev - best.ev) / max(temp, 1))
		weights = append(weights, w)
		sum += w
	}
	x := r.Float64() * sum
	for i, w := range weights {
		if x < w {
			return opts[i].a
		}
		x -= w
	}
	return best.a
}

// contOf is how likely each hand in the pool is to call and to raise a bet
// of x times the pot, for an opponent whose range is w. They defend the top
// of their range as often as they've been seen to against bets, less against
// bigger ones, and never with nothing. Draws count as stronger than they are.
// The top of what they defend raises, more for aggressive players.
func contOf(made []float64, draws []draw, w []float64, x float64, river bool, t tendencies) (call, raise []float64) {
	edge := make([]float64, len(made))
	for i, m := range made {
		edge[i] = m
		switch {
		case river:
		case draws[i] >= openEnded:
			edge[i] += 0.2
		case draws[i] == gutshot:
			edge[i] += 0.08
		}
	}
	// against a bet of about 60% of the pot they defend 1-t.fold
	defend := min(max((1-t.fold)*1.6/(1+x), 0.05), 1)
	raises := min(0.12*t.aggro, 0.3)
	cut, top := cutoff(edge, w, defend), cutoff(edge, w, defend*raises)
	call, raise = make([]float64, len(made)), make([]float64, len(made))
	for i, e := range edge {
		c := 1 / (1 + math.Exp(-(e-cut)/0.02)) / (1 + math.Exp(-(e-0.3)/0.04))
		rz := c / (1 + math.Exp(-(e-top)/0.02))
		call[i], raise[i] = c-rz, rz
	}
	return call, raise
}

// cutoff is the edge that the top share of w's weight is above.
func cutoff(edge, w []float64, share float64) float64 {
	if share >= 1 {
		return 0
	}
	idx := make([]int, len(edge))
	total := 0.0
	for i := range idx {
		idx[i] = i
		total += w[i]
	}
	slices.SortFunc(idx, func(a, b int) int { return cmp.Compare(edge[b], edge[a]) })
	kept := 0.0
	for _, i := range idx {
		kept += w[i]
		if kept >= share*total {
			return edge[i]
		}
	}
	return 0
}

// share is how much of ws's weight cont keeps.
func share(w, cont []float64) float64 {
	var kept, all float64
	for i, x := range w {
		kept += x * cont[i]
		all += x
	}
	return kept / max(all, 1e-9)
}

func hardPostflop(s spot, p *pool, ws map[int][]float64, r *rand.Rand) Action {
	owe := s.top - s.bet
	river := len(s.board) == 5
	opp, players := opponents(s, ws, nil)
	made, draws := p.strength(len(s.board))
	pot := float64(s.pot)

	// bet sizes, as a share of the pot after calling
	sizes := []float64{0.33, 0.7}
	switch {
	case river && s.game.Pot:
		sizes = []float64{0.5, 0.8, 1}
	case river:
		sizes = []float64{0.5, 0.8, 1.4} // an overbet, polar
	case len(s.board) == 4:
		sizes = []float64{0.5, 0.8}
	}
	type bet struct {
		add           uint // chips the bot puts in
		x             float64
		calls, raises [][]float64 // each opponent's chance to call or raise, by hand
		fold, raised  float64     // chance everyone folds, and someone raises
		callers       float64     // expected callers when nobody raises
	}
	var bets []bet
	seen := map[uint]bool{}
	for _, x := range sizes {
		a := s.size(owe + uint(x*float64(s.pot+owe)))
		if a.Kind != Raise || seen[a.Amount] {
			continue
		}
		seen[a.Amount] = true
		b := bet{add: a.Amount, x: float64(a.Amount-owe) / float64(s.pot+owe), fold: 1}
		calm := 1.0 // chance nobody raises
		for k, pl := range players {
			c, rz := contOf(made, draws, opp[k], b.x, river, tendenciesOf(s.reads[pl.Seat]))
			if pl.Stack == 0 || a.Amount >= s.stack {
				// all in already, or can't be raised: calls or folds
				for i := range c {
					c[i], rz[i] = c[i]+rz[i], 0
				}
				if pl.Stack == 0 {
					for i := range c {
						c[i] = 1
					}
				}
			}
			b.calls, b.raises = append(b.calls, c), append(b.raises, rz)
			stay, raise := share(opp[k], c), share(opp[k], rz)
			b.fold *= 1 - stay - raise
			calm *= 1 - raise
			b.callers += stay
		}
		b.raised = 1 - calm
		b.callers /= max(1-b.fold-b.raised, 1e-9)
		bets = append(bets, b)
	}
	// equity when called, then when raised, for each bet
	var conts [][][]float64
	for _, b := range bets {
		conts = append(conts, b.calls, b.raises)
	}
	trials := 700
	if s.game.HoleCards() > 2 {
		trials = 400
	}
	e, eCall := p.odds(s.hole, opp, conts, trials, r)

	// how much of its equity the bot gets to keep by the river, in position
	// or not, and a little more for a draw's implied odds
	realize := 1.0
	if !river {
		realize = 0.85
		if s.behind == 0 {
			realize = 0.95
		}
	}
	implied := 0.0
	if d := drawOf(s.hole, s.board); !river && d >= openEnded && s.stack > s.pot {
		implied = 0.04
	}
	why := fmt.Sprintf("equity %.2f", e)
	var opts []option
	if owe == 0 {
		opts = append(opts, option{because(Action{Kind: Check}, "%s, check", why), e * realize * pot})
	} else {
		call := e*realize + implied
		opts = append(opts,
			option{because(Action{Kind: Fold}, "%s, fold", why), 0},
			option{because(Action{Kind: Call}, "%s, call %d into %d", why, owe, s.pot), call*(pot+float64(owe)) - float64(owe)})
	}
	for j, b := range bets {
		// when called, the pot has the bet, the calls, and what's already in;
		// with the initiative the bot keeps a little more of its equity
		add := float64(b.add)
		matched := add - float64(owe)
		after := pot + add + b.callers*matched
		keep := min(realize+0.05, 1)
		called := eCall[2*j]*keep*after - add
		// raised, it calls about the pot again or folds what it bet
		eRaised := eCall[2*j+1]
		reraised := max(-add, eRaised*keep*3*after-add-after)
		ev := b.fold*pot + max(1-b.fold-b.raised, 0)*called + b.raised*reraised
		ev += s.style.aggro * 0.02 * pot
		opts = append(opts, option{because(Action{Kind: Raise, Amount: b.add}, "%s, %.2f called, %.2f raised, bet %.0f%% pot, folds %.0f%%, raised %.0f%%", why, eCall[2*j], eRaised, b.x*100, b.fold*100, b.raised*100), ev})
	}
	return pick(opts, pot, r)
}

// pushWidth is the share of hands to move all in with, unopened, with eff
// big blinds and behind players left to act.
func pushWidth(eff float64, behind, dealt int) float64 {
	base := openWidth(behind, dealt) * 1.3
	if dealt == 2 {
		base = 0.75
	}
	return min(base*math.Pow(10/max(eff, 1), 0.7), 0.95)
}

func hardPreflop(s spot, p *pool, ws map[int][]float64, r *rand.Rand) Action {
	pct := percentile(s.game, s.hole) + 0.01*s.style.loose
	owe := s.top - s.bet
	dealt, behind := 0, 0
	eff := 0.0 // the most chips the bot can win or lose, in big blinds
	raises := 0
	for _, a := range s.line {
		if a.Street == holdem.Preflop && a.Kind == holdem.KindRaise {
			raises++
		}
	}
	for _, pl := range s.view.Players {
		if pl.Dealt {
			dealt++
		}
		if pl.In && pl.Seat != s.seat {
			eff = max(eff, float64(min(pl.Stack+pl.Bet, s.stack+s.bet)))
			if !pl.Acted && pl.Stack > 0 {
				behind++
			}
		}
	}
	eff /= float64(s.bb)
	noLimit := !s.game.Pot && s.game.HoleCards() == 2
	allIn := s.size(s.stack)

	if raises == 0 {
		width := openWidth(behind, dealt) * wider(s.game)
		limpers := s.limpers
		if limpers > 0 {
			width *= 0.7
		}
		if noLimit && eff <= 14 {
			if pct >= 1-pushWidth(eff, behind, dealt) {
				return because(allIn, "%.2f, push %.0f bb", pct, eff)
			}
		} else if pct >= 1-width+0.02*(r.Float64()-0.5) {
			to := s.bb*5/2 + s.bb*uint(limpers)
			if s.sb {
				to = 3 * s.bb
			}
			return because(s.size(jitter(to, s.bb, r)-s.bet), "%.2f in the top %.0f%%, open", pct, width*100)
		}
		if owe == 0 {
			return because(Action{Kind: Check}, "%.2f, check", pct)
		}
		return because(Action{Kind: Fold}, "%.2f below the top %.0f%%, fold", pct, width*100)
	}

	// against the players who've chosen to put money in
	voluntary := map[int]bool{}
	for _, a := range s.line {
		if a.Kind == holdem.KindCall || a.Kind == holdem.KindRaise {
			voluntary[a.Seat] = true
		}
	}
	opp, _ := opponents(s, ws, func(pl holdem.Player) bool { return voluntary[pl.Seat] })
	e, _ := p.odds(s.hole, opp, nil, 500, r)
	pot := float64(s.pot)
	price := float64(owe) / (pot + float64(owe))
	realize := 0.82
	if behind == 0 && !s.bbs && !s.sb {
		realize = 0.95 // in position
	}
	// against a stronger range each raise, with less behind
	realize -= 0.1 * float64(raises-1)
	if s.game.HoleCards() > 2 {
		realize = 0.9 // Omaha equities run close
	}
	hi, lo, suited := classOf(s.hole)
	if len(s.hole) == 2 && (hi == lo || suited) && eff > 40 {
		realize += 0.05 // sets and flushes get paid
	}
	why := fmt.Sprintf("%.2f, equity %.2f against %d raises, price %.2f", pct, e, raises, price)
	callOK := e*realize >= price
	short := noLimit && eff <= 20 || s.stack <= owe*3

	switch {
	case short:
		// all in or fold
		risk := float64(s.stack)
		if e*(pot+2*risk-float64(owe)) >= risk {
			return because(allIn, "%s, all in", why)
		}
	case raises == 1:
		value := pct >= 0.965 || e >= 0.56+0.04*float64(len(opp)-1)
		bluff := len(opp) == 1 && suitedPlayable(s.hole) && pct >= 0.55 && s.often(0.3, r)
		if value || bluff {
			mult := uint(3)
			if behind > 0 || s.bbs || s.sb {
				mult = 4 // out of position
			}
			return because(s.size(jitter(mult*s.top, s.bb, r)-s.bet), "%s, 3-bet (bluff %v)", why, bluff && !value)
		}
	case raises == 2:
		bluff := len(s.hole) == 2 && hi == 12 && suited && lo <= 3 && s.often(0.15, r)
		if pct >= 0.99 || e >= 0.6 || bluff {
			return because(s.size(jitter(s.top*9/4, s.bb, r)-s.bet), "%s, 4-bet (bluff %v)", why, bluff && e < 0.6)
		}
		callOK = callOK && e*realize >= price+0.03
	default:
		if e >= 0.52 {
			return because(allIn, "%s, all in", why)
		}
	}
	if callOK {
		return because(Action{Kind: Call}, "%s, call", why)
	}
	if owe == 0 {
		return because(Action{Kind: Check}, "%s, check", why)
	}
	return because(Action{Kind: Fold}, "%s, fold", why)
}
