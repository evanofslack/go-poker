package bot

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// The sims play bots against each other on the real engine and log how each
// one plays, like a poker tracker would. Shuffles aren't seeded and they take
// a while, so they only run with BOTSIM=1. With BOTSIM_LOG=<dir> they also
// write every hand, with each bot's cards and reasons, to <dir>/<test>.log.

const simBB = 20

// TestSim plays medium against easy heads up and checks medium comes out ahead.
func TestSim(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
	const hands = 2000
	res := sim(t, holdem.Game{}, []string{Medium, Easy}, hands, 100*simBB, false)
	report(t, res)
	if res[0].net <= 0 {
		t.Errorf("medium lost %d chips to easy over %d hands", -res[0].net, hands)
	}
}

// TestSimBaseline seats three medium bots with three playing as medium did
// before its rewrite, and checks the new one comes out ahead.
func TestSimBaseline(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
	levels := []string{Medium, baselineLevel, Medium, baselineLevel, Medium, baselineLevel}
	res := sim(t, holdem.Game{}, levels, 4000, 100*simBB, false)
	report(t, res)
	net := 0
	for _, s := range res {
		if s.level == Medium {
			net += s.net
		}
	}
	if net <= 0 {
		t.Errorf("medium lost %d chips to baseline", -net)
	}
}

// TestSimTable plays six bots of a level at one table, the lineup a person
// joining a bot table meets.
func TestSimTable(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
	for _, level := range []string{Medium, Easy} {
		t.Run(level, func(t *testing.T) {
			levels := make([]string, 6)
			for i := range levels {
				levels[i] = level
			}
			report(t, sim(t, holdem.Game{}, levels, 2000, 100*simBB, false))
		})
	}
}

// TestBotsPlayOmaha plays PLO and PLO5, and checks bots only make legal moves.
func TestBotsPlayOmaha(t *testing.T) {
	for _, g := range []holdem.Game{{Hole: 4, Pot: true}, {Hole: 5, Pot: true}, {Hole: 4}} {
		sim(t, g, []string{Medium, Easy, Medium, Easy}, 50, 100*simBB, false)
	}
}

// TestSimOmaha plays medium against easy at PLO heads up and checks medium
// comes out ahead.
func TestSimOmaha(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
	const hands = 2000
	res := sim(t, holdem.Game{Hole: 4, Pot: true}, []string{Medium, Easy}, hands, 100*simBB, false)
	report(t, res)
	if res[0].net <= 0 {
		t.Errorf("medium lost %d chips to easy at PLO over %d hands", -res[0].net, hands)
	}
}

// TestBotsFacingStraddles plays six bots with every straddle posted, and
// checks they only make legal moves.
func TestBotsFacingStraddles(t *testing.T) {
	sim(t, holdem.Game{}, []string{Medium, Easy, Medium, Easy, Medium, Easy}, 300, 100*simBB, true)
}

// stats is how one seat played over a sim.
type stats struct {
	level string
	net   int
	hands int

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

// sim plays game between bots of levels, in seats from 1, rebuying busted
// ones, and returns how each one played. straddle posts every straddle there
// can be.
func sim(t *testing.T, game holdem.Game, levels []string, hands int, buyIn uint, straddle bool) []stats {
	g := holdem.New(holdem.Config{Game: game, SmallBlind: simBB / 2, BigBlind: simBB}, [32]byte{byte(rand.Uint64())})
	res := make([]stats, len(levels))
	bought := make([]uint, len(levels))
	for i, level := range levels {
		must(t, g.Sit(i+1, buyIn))
		res[i].level, bought[i] = level, buyIn
	}
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	name := func(seat int) string { return fmt.Sprintf("%s%d", levels[seat-1], seat) }
	log := simLog(t)

	flags := make([]handFlags, len(levels))
	var board []eval.Card
	flopped := false
	// who made the last bet or raise before this street, and on it
	aggressor, latest := 0, 0
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
					fmt.Fprintf(log, "  %-9s %5d  %s\n", name(seat), flags[seat-1].start, cardList(p.Cards[:]))
				}
				board, flopped, aggressor, latest = nil, false, 0, 0
			case holdem.Acted:
				if e.Kind == holdem.KindBet || e.Kind == holdem.KindRaise {
					latest = e.Seat
				}
			case holdem.StreetDealt:
				if e.Run > 0 {
					continue
				}
				board = append(board, e.Cards...)
				flopped = true
				if latest != 0 {
					aggressor, latest = latest, 0
				}
				fmt.Fprintf(log, "  board %s\n", cardList(board))
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
				fmt.Fprintf(log, "  %s shows %s\n", name(e.Seat), cardList(e.Cards[:]))
			case holdem.HandEnded:
				ended = true
			}
		}
		return ended
	}

	for played := 0; played < hands; {
		var events []holdem.Event
		var err error
		switch {
		case !g.InHand():
			for i, p := range g.Players {
				if p.Stack == 0 {
					must(t, g.AddChips(p.Seat, buyIn))
					must(t, g.SetSittingOut(p.Seat, false))
					bought[i] += buyIn
				}
			}
			n := 0
			if straddle {
				n = len(g.Positions().Straddle)
			}
			events, err = g.StartHand(holdem.Deal{Straddles: n})
			must(t, err)
		case !g.Betting:
			events, err = g.NextStreet()
			must(t, err)
		default:
			seat := g.ToAct
			view := g.View(seat)
			s := spotOf(view, seat, aggressor)
			s.style = styleOf(name(seat))
			a := decide(levels[seat-1], s, r)
			note(&res[seat-1], &flags[seat-1], view, seat, a)
			fmt.Fprintf(log, "    %-7s %-9s %-5s %-5s %5s  %s\n", g.Street, name(seat), cardList(s.hole[:]), kindNames[a.Kind], amount(a), a.Reason)
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
			if f.start == 0 {
				continue // not dealt in
			}
			s := &res[i]
			s.hands++
			s.vpip += b2i(f.vpip)
			s.pfr += b2i(f.pfr)
			s.reraiseSpots += b2i(f.reraiseSpot)
			s.bets3 += b2i(f.reraised)
			s.sawFlop += b2i(flopped && !f.foldedPreflop)
			s.showdowns += b2i(f.showdown)
			s.wonSD += b2i(f.showdown && f.won)
			if net := int(g.Players[i].Stack) - int(f.start); net != 0 {
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
// for comparison.
func report(t *testing.T, res []stats) {
	t.Helper()
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
