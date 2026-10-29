package bot

import (
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// The sims play bots against each other on the real engine and log how each
// one plays, like a poker tracker would. They take a while, so they only run
// with BOTSIM=1. With BOTSIM_LOG=<dir> they also write every hand, with each
// bot's cards and reasons, to <dir>/<test>.log.
//
// Matches are duplicate: the same cards are played twice with the lineup
// swapped, so luck mostly cancels and a few thousand hands tell bots apart.

const simBB = 20

// games are the games the sims play, by name.
var games = []struct {
	name string
	game holdem.Game
	bomb bool // every hand a double board bomb pot
}{
	{"nlh", holdem.Game{}, false},
	{"plo", holdem.Game{Hole: 4, Pot: true}, false},
	{"short", holdem.Game{Short: true}, false},
	{"bomb", holdem.Game{}, true},
}

// TestSim plays medium against easy heads up and checks medium comes out
// ahead in every game.
func TestSim(t *testing.T) {
	simOnly(t)
	for _, g := range games {
		t.Run(g.name, func(t *testing.T) {
			m := versus(t, table{game: g.game, bombs: g.bomb, hands: 1000}, Medium, Easy, 2)
			if m.lo() <= 0 {
				t.Errorf("medium against easy: %v", m)
			}
		})
	}
}

// TestSimHard plays hard against medium in every game and against each
// exploiter at no limit, heads up and six handed, and checks it comes out
// ahead. It takes about an hour on ten cores.
func TestSimHard(t *testing.T) {
	simOnly(t)
	type duelOf struct {
		game     int
		opponent string
	}
	var duels []duelOf
	for i := range games {
		duels = append(duels, duelOf{i, Medium})
	}
	for _, x := range []string{station, jammer, nit, trapper} {
		duels = append(duels, duelOf{0, x})
	}
	for _, d := range duels {
		g := games[d.game]
		for _, seats := range []int{2, 6} {
			t.Run(fmt.Sprintf("%s/%s/%d", g.name, d.opponent, seats), func(t *testing.T) {
				var mu sync.Mutex
				var all []float64
				t.Run("shards", func(t *testing.T) {
					for k := range 10 {
						t.Run(strconv.Itoa(k), func(t *testing.T) {
							t.Parallel()
							chips := duel(t, table{game: g.game, bombs: g.bomb, hands: 500}, Hard, d.opponent, seats)
							mu.Lock()
							all = append(all, chips...)
							mu.Unlock()
						})
					}
				})
				m := matchOf(all, seats/2)
				t.Logf("hard against %s: %v", d.opponent, m)
				if m.lo() <= 0 {
					t.Errorf("hard against %s at %s, %d seats: %v", d.opponent, g.name, seats, m)
				}
			})
		}
	}
}

// TestSimBaseline seats three medium bots with three playing as medium did
// before its rewrite, and checks the new one comes out ahead.
func TestSimBaseline(t *testing.T) {
	simOnly(t)
	if m := versus(t, table{hands: 1500}, Medium, baselineLevel, 6); m.mean <= 0 {
		t.Errorf("medium against baseline: %v", m)
	}
}

// TestSimVersus plays the first of BOTSIM_LEVELS (medium,easy by default)
// against each of the others in BOTSIM_GAMES (all by default), heads up and
// six handed, BOTSIM_HANDS hands in each of BOTSIM_SHARDS shards at once.
func TestSimVersus(t *testing.T) {
	simOnly(t)
	levels := strings.Split(or(os.Getenv("BOTSIM_LEVELS"), Medium+","+Easy), ",")
	only := os.Getenv("BOTSIM_GAMES")
	hands, err := strconv.Atoi(or(os.Getenv("BOTSIM_HANDS"), "1500"))
	must(t, err)
	shards, err := strconv.Atoi(or(os.Getenv("BOTSIM_SHARDS"), "1"))
	must(t, err)
	a := levels[0]
	for _, g := range games {
		if only != "" && !strings.Contains(only, g.name) {
			continue
		}
		for _, b := range levels[1:] {
			for _, seats := range []int{2, 6} {
				var mu sync.Mutex
				var all []float64
				t.Run(fmt.Sprintf("%s/%s-%s/%d", g.name, a, b, seats), func(t *testing.T) {
					for k := range shards {
						t.Run(strconv.Itoa(k), func(t *testing.T) {
							t.Parallel()
							chips := duel(t, table{game: g.game, bombs: g.bomb, hands: hands}, a, b, seats)
							mu.Lock()
							all = append(all, chips...)
							mu.Unlock()
						})
					}
				})
				t.Logf("%s %s against %s, %d seats, %d hands twice: %v", g.name, a, b, seats, len(all), matchOf(all, seats/2))
			}
		}
	}
}

// TestSimTable plays six bots of a level at one table, the lineup a person
// joining a bot table meets.
func TestSimTable(t *testing.T) {
	simOnly(t)
	for _, level := range []string{Medium, Easy} {
		t.Run(level, func(t *testing.T) {
			levels := make([]string, 6)
			for i := range levels {
				levels[i] = level
			}
			report(t, sim(t, table{levels: levels, hands: 2000}))
		})
	}
}

// TestBotsPlayEveryGame plays a few hands of every game, with straddles, bomb
// pots and double boards, and checks bots only make legal moves.
func TestBotsPlayEveryGame(t *testing.T) {
	levels := []string{Hard, Medium, Easy, Hard, Medium, Easy}
	for _, g := range []holdem.Game{{}, {Hole: 4, Pot: true}, {Hole: 5, Pot: true}, {Hole: 4}, {Short: true}, {Hole: 4, Pot: true, Short: true}} {
		sim(t, table{game: g, levels: levels[:4], hands: 30})
		sim(t, table{game: g, levels: levels[:4], hands: 8, bombs: true})
	}
	sim(t, table{levels: levels, hands: 200, straddle: true})
}

func simOnly(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
}

func or(s, otherwise string) string {
	if s == "" {
		return otherwise
	}
	return s
}

// table is a sim to play.
type table struct {
	game     holdem.Game
	levels   []string // by seat, from 1
	hands    int
	straddle bool   // post every straddle there can be
	bombs    bool   // every hand a double board bomb pot of a big blind
	seed     uint64 // the cards, 0 for any
	reset    bool   // every stack back to 100 big blinds before each hand
}

// match is how one level did against another, in big blinds per 100 hands
// a seat, with a 95% interval.
type match struct {
	mean, ci float64
}

func (m match) lo() float64 { return m.mean - m.ci }

func (m match) String() string { return fmt.Sprintf("%+.1f ± %.1f bb/100", m.mean, m.ci) }

// versus plays a against b at seats seats, a and b taking turns around the
// table, then again on the same cards with a and b swapped, and returns how
// a did.
func versus(t *testing.T, tb table, a, b string, seats int) match {
	t.Helper()
	m := matchOf(duel(t, tb, a, b, seats), seats/2)
	t.Logf("%s against %s, %d seats, %d hands twice: %v", a, b, seats, tb.hands, m)
	return m
}

// duel plays a against b at seats seats, a and b taking turns around the
// table, then again on the same cards with a and b swapped, and returns a's
// chips each hand over both, so the same cards pair up.
func duel(t *testing.T, tb table, a, b string, seats int) []float64 {
	t.Helper()
	tb.seed, tb.reset = rand.Uint64()|1, true
	var runs [2][]stats
	for k := range runs {
		tb.levels = make([]string, seats)
		for i := range tb.levels {
			tb.levels[i] = a
			if (i+k)%2 == 1 {
				tb.levels[i] = b
			}
		}
		runs[k] = sim(t, tb)
		report(t, runs[k])
	}
	sum := make([]float64, tb.hands)
	for _, run := range runs {
		for _, s := range run {
			if s.level != a {
				continue
			}
			for i, net := range s.nets {
				sum[i] += float64(net)
			}
		}
	}
	return sum
}

// matchOf is how a level did from its chips each hand over two runs, with
// seats of it in each.
func matchOf(sum []float64, seats int) match {
	var mean, sq float64
	for _, x := range sum {
		mean += x
	}
	mean /= float64(len(sum))
	for _, x := range sum {
		sq += (x - mean) * (x - mean)
	}
	sd := math.Sqrt(sq / float64(len(sum)-1))
	// per seat, per 100 hands, in big blinds
	scale := 100 / float64(2*seats) / simBB
	return match{mean: mean * scale, ci: 1.96 * sd / math.Sqrt(float64(len(sum))) * scale}
}

// stats is how one seat played over a sim.
type stats struct {
	level string
	net   int
	hands int
	nets  []int // chips won or lost each hand

	vpip, pfr           int // hands it called or raised preflop, and raised
	reraiseSpots, bets3 int // hands it faced a preflop raise, and reraised
	sawFlop             int
	showdowns, wonSD    int
	aggressive, calls   int // postflop bets and raises, and calls
	facedBet, foldBet   int // postflop decisions facing a bet, and folds to one
	sizes               [len(sizeNames)]int
}

// sizeNames buckets postflop bets and raises by how much of the pot they add
var sizeNames = [...]string{"<=40%", "<=60%", "<=80%", "<=110%", "over", "all in"}

// handFlags is what a seat did in the current hand.
type handFlags struct {
	vpip, pfr, reraiseSpot, reraised bool
	foldedPreflop                    bool
	showdown, won                    bool
	start                            uint // stack before the blinds, 0 if not dealt in
}

// sim plays tb, rebuying busted bots, and returns how each one played.
func sim(t *testing.T, tb table) []stats {
	seed := [32]byte{}
	s := tb.seed
	if s == 0 {
		s = rand.Uint64()
	}
	for i := range 8 {
		seed[i] = byte(s >> (8 * i))
	}
	buyIn := uint(100 * simBB)
	g := holdem.New(holdem.Config{Game: tb.game, SmallBlind: simBB / 2, BigBlind: simBB}, seed)
	res := make([]stats, len(tb.levels))
	bought := make([]uint, len(tb.levels))
	reads := map[int]Read{}
	for i, level := range tb.levels {
		must(t, g.Sit(i+1, buyIn))
		res[i].level, bought[i] = level, buyIn
	}
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	name := func(seat int) string { return fmt.Sprintf("%s%d", tb.levels[seat-1], seat) }
	log := simLog(t)

	flags := make([]handFlags, len(tb.levels))
	var line []holdem.Acted
	var top uint           // this street's biggest bet
	bets := map[int]uint{} // and each seat's
	flopped := false
	// see follows what events did to the hand, and reports whether it ended
	see := func(events []holdem.Event) bool {
		ended := false
		for _, e := range events {
			switch e := e.(type) {
			case holdem.HandStarted:
				fmt.Fprintf(log, "\nhand %d, %s deals\n", e.Hand, name(e.Button))
				for i := range flags {
					flags[i] = handFlags{}
				}
				for _, seat := range e.Seats {
					p := g.Player(seat)
					flags[seat-1].start = p.Stack + p.Committed
					fmt.Fprintf(log, "  %-9s %5d  %s\n", name(seat), flags[seat-1].start, cardList(p.Cards))
				}
				line, flopped, top = nil, false, 0
				clear(bets)
			case holdem.Acted:
				rd := reads[e.Seat]
				if len(line) > 0 && line[len(line)-1].Street != e.Street {
					top = 0
					clear(bets)
				}
				if e.Street != holdem.Preflop && e.Kind != holdem.KindBomb && e.Kind != holdem.KindAnte && top > bets[e.Seat] {
					rd.FacedBets++
					rd.FoldedToBets += b2i(e.Kind == holdem.KindFold)
				}
				if e.Amount > 0 {
					bets[e.Seat] = e.To
					top = max(top, e.To)
				}
				line = append(line, e)
				switch e.Kind {
				case holdem.KindBet, holdem.KindRaise:
					rd.Aggressive++
				case holdem.KindCall:
					rd.Calls++
				}
				reads[e.Seat] = rd
			case holdem.StreetDealt:
				if e.Run > 0 {
					continue
				}
				flopped = true
				fmt.Fprintf(log, "  board %s\n", cardList(g.Board))
				if g.Double && len(g.Boards) > 1 {
					fmt.Fprintf(log, "  board %s\n", cardList(g.Boards[1]))
				}
			case holdem.PotWon:
				var names []string
				for _, seat := range e.Winners {
					flags[seat-1].won = true
					names = append(names, name(seat))
				}
				fmt.Fprintf(log, "  %s wins %d %s\n", strings.Join(names, ", "), e.Amount, e.Hand)
				if e.Hand != "" {
					for _, seat := range g.Pots[e.Pot].Eligible {
						flags[seat-1].showdown = true
					}
				}
			case holdem.Shown:
				fmt.Fprintf(log, "  %s shows %s\n", name(e.Seat), cardList(e.Cards))
			case holdem.HandEnded:
				ended = true
			}
		}
		return ended
	}

	for played := 0; played < tb.hands; {
		var events []holdem.Event
		var err error
		switch {
		case !g.InHand():
			for i, p := range g.Players {
				switch {
				case tb.reset && p.Stack > buyIn:
					must(t, g.RemoveChips(p.Seat, p.Stack-buyIn))
					bought[i] -= p.Stack - buyIn
				case tb.reset && p.Stack < buyIn, p.Stack == 0:
					bought[i] += buyIn - p.Stack
					must(t, g.AddChips(p.Seat, buyIn-p.Stack))
					must(t, g.SetSittingOut(p.Seat, false))
				}
			}
			d := holdem.Deal{}
			if tb.straddle {
				d.Straddles = len(g.Positions().Straddle)
			}
			if tb.bombs {
				d.Bomb, d.Boards = simBB, 2
			}
			events, err = g.StartHand(d)
			must(t, err)
		case !g.Betting:
			events, err = g.NextStreet()
			must(t, err)
		default:
			seat := g.ToAct
			view := g.View(seat)
			s := spotOf(view, seat, line)
			s.style = styleOf(name(seat))
			s.reads = map[int]Read{}
			for k, rd := range reads {
				if k != seat {
					s.reads[k] = rd
				}
			}
			a := decide(tb.levels[seat-1], s, r)
			note(&res[seat-1], &flags[seat-1], view, seat, a)
			fmt.Fprintf(log, "    %-7s %-9s %-5s %-5s %5s  %s\n", g.Street, name(seat), cardList(s.hole), kindNames[a.Kind], amount(a), a.Reason)
			events, err = g.Act(seat, toHoldem(s, a))
			if err != nil {
				t.Fatalf("%s's %+v rejected: %v", name(seat), a, err)
			}
		}
		if !see(events) {
			continue
		}
		played++
		for i, f := range flags {
			s := &res[i]
			if f.start == 0 {
				s.nets = append(s.nets, 0)
				continue // not dealt in
			}
			net := int(g.Players[i].Stack) - int(f.start)
			s.nets = append(s.nets, net)
			s.hands++
			s.vpip += b2i(f.vpip)
			s.pfr += b2i(f.pfr)
			s.reraiseSpots += b2i(f.reraiseSpot)
			s.bets3 += b2i(f.reraised)
			s.sawFlop += b2i(flopped && !f.foldedPreflop)
			s.showdowns += b2i(f.showdown)
			s.wonSD += b2i(f.showdown && f.won)
			rd := reads[i+1]
			rd.Hands++
			rd.Bombs += b2i(tb.bombs)
			rd.VPIP += b2i(f.vpip)
			rd.PFR += b2i(f.pfr)
			rd.ThreeBetSpots += b2i(f.reraiseSpot)
			rd.ThreeBets += b2i(f.reraised)
			reads[i+1] = rd
			if net != 0 {
				fmt.Fprintf(log, "  %-9s %+d\n", name(i+1), net)
			}
		}
	}

	for i, p := range g.Players {
		res[i].net = int(p.Stack) - int(bought[i])
	}
	return res
}

// note counts seat's move a. view is from before it.
func note(s *stats, f *handFlags, view holdem.View, seat int, a Action) {
	var top, pot, bet, stack uint
	for _, p := range view.Players {
		top = max(top, p.Bet)
		pot += p.Committed
		if p.Seat == seat {
			bet, stack = p.Bet, p.Stack
		}
	}
	owe := top - bet
	if view.Street == holdem.Preflop {
		raised := top > view.Config.BigBlind<<len(view.Straddles)
		f.reraiseSpot = f.reraiseSpot || raised
		switch a.Kind {
		case Call:
			f.vpip = true
		case Raise:
			f.vpip, f.pfr = true, true
			f.reraised = f.reraised || raised
		case Fold:
			f.foldedPreflop = true
		}
		return
	}
	if owe > 0 {
		s.facedBet++
		if a.Kind == Fold {
			s.foldBet++
		}
	}
	switch a.Kind {
	case Call:
		s.calls++
	case Raise:
		s.aggressive++
		s.sizes[sizeOf(a.Amount, owe, pot, stack)]++
	}
}

// sizeOf buckets a bet or raise of amount chips by how much it adds to the
// pot after calling
func sizeOf(amount, owe, pot, stack uint) int {
	if amount >= stack {
		return len(sizeNames) - 1
	}
	frac := float64(amount-owe) / float64(pot+owe)
	for i, limit := range []float64{0.4, 0.6, 0.8, 1.1} {
		if frac <= limit {
			return i
		}
	}
	return len(sizeNames) - 2
}

// report logs a row of stats for each seat, then a solid human 6-max player's
// for comparison, when the test is verbose.
func report(t *testing.T, res []stats) {
	t.Helper()
	if !testing.Verbose() {
		return
	}
	t.Logf("%-9s %6s %5s %5s %5s %5s %5s %5s %5s %7s   postflop sizes %v",
		"seat", "hands", "vpip", "pfr", "3bet", "af", "fold", "wtsd", "w$sd", "bb/100", sizeNames)
	for i, s := range res {
		sizes := make([]string, len(s.sizes))
		for k, n := range s.sizes {
			sizes[k] = pct(n, s.aggressive)
		}
		t.Logf("%-9s %6d %5s %5s %5s %5.1f %5s %5s %5s %+7.1f   %s",
			fmt.Sprintf("%s%d", s.level, i+1), s.hands,
			pct(s.vpip, s.hands), pct(s.pfr, s.hands), pct(s.bets3, s.reraiseSpots),
			float64(s.aggressive)/float64(max(s.calls, 1)),
			pct(s.foldBet, s.facedBet), pct(s.showdowns, s.sawFlop), pct(s.wonSD, s.showdowns),
			float64(s.net)/simBB/float64(max(s.hands, 1))*100, strings.Join(sizes, " "))
	}
	t.Logf("%-9s %6s %5s %5s %5s %5s %5s %5s %5s", "human", "", "22-28", "17-22", "6-9", "2-3", "40-55", "25-30", "50-55")
}

func pct(n, of int) string {
	if of == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", float64(n)/float64(of)*100)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

var kindNames = map[Kind]string{Check: "check", Call: "call", Raise: "raise", Fold: "fold"}

func amount(a Action) string {
	if a.Kind != Raise {
		return ""
	}
	return fmt.Sprint(a.Amount)
}

func cardList(cards []eval.Card) string {
	var names []string
	for _, c := range cards {
		if c != 0 {
			names = append(names, c.String())
		}
	}
	return strings.Join(names, " ")
}

// simLog is where the test writes its hands: a file under BOTSIM_LOG, or nowhere.
func simLog(t *testing.T) io.Writer {
	dir := os.Getenv("BOTSIM_LOG")
	if dir == "" {
		return io.Discard
	}
	f, err := os.Create(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "-")+".log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
