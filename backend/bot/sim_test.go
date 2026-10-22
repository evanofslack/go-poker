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
	"github.com/evanofslack/go-poker/poker"
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
	res := sim(t, []string{Medium, Easy}, hands, 100*simBB)
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
	res := sim(t, levels, 4000, 100*simBB)
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
			report(t, sim(t, levels, 2000, 100*simBB))
		})
	}
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

// handFlags is what a seat did preflop in the current hand.
type handFlags struct {
	vpip, pfr, reraiseSpot, reraised bool
	foldedPreflop                    bool
	start                            uint // stack before the blinds
}

// sim plays hands between bots of levels, rebuying busted ones, and returns
// how each one played.
func sim(t *testing.T, levels []string, hands int, buyIn uint) []stats {
	g := poker.NewGame(poker.GameConfig{MaxBuy: buyIn, BigBlind: simBB, SmallBlind: simBB / 2})
	res := make([]stats, len(levels))
	for i, level := range levels {
		pn := g.AddPlayer()
		must(t, poker.BuyIn(g, pn, buyIn))
		must(t, poker.ToggleReady(g, pn, 0))
		must(t, poker.SetSeatID(g, pn, uint(i+1)))
		res[i].level = level
	}
	// players stay in seat order, so the index is the level's index
	must(t, g.Start()) // deals the first hand
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	name := func(pn uint) string { return fmt.Sprintf("%s%d", levels[pn], pn+1) }
	log := simLog(t)

	flags := make([]handFlags, len(levels))
	startHand(g.GenerateOmniView(), 1, flags, log, name)
	shown := 0 // board cards logged
	// the engine deals each street itself, and stops betting once the hand
	// is over
	for played := 0; ; {
		view := g.GenerateOmniView()
		if !view.Betting {
			finishHand(g, view, res, flags, log, name)
			if played++; played == hands {
				return totals(g, res)
			}
			for pn, p := range view.Players {
				if p.Stack == 0 {
					must(t, poker.BuyIn(g, uint(pn), buyIn))
					if !p.Ready {
						must(t, poker.ToggleReady(g, uint(pn), 0))
					}
				}
			}
			must(t, poker.Deal(g, view.DealerNum, 0))
			startHand(g.GenerateOmniView(), played+1, flags, log, name)
			shown = 0
			continue
		}
		if b := dealt(view.CommunityCards); len(b) != shown {
			shown = len(b)
			fmt.Fprintf(log, "  board %s\n", cardList(b))
		}
		pn := view.ActionNum
		p := view.Players[pn]
		pv := g.GeneratePlayerView(pn)
		a := decide(levels[pn], pv, pn, r)
		note(&res[pn], &flags[pn], view, pn, a)
		fmt.Fprintf(log, "    %-7s %-8s %-5s %-5s %5s  %s\n", streetName(view), name(pn), cardList(pv.Players[pn].Cards[:]), kindNames[a.Kind], amount(a), a.Reason)

		var top uint
		for _, q := range view.Players {
			top = max(top, q.Bet)
		}
		var err error
		switch a.Kind {
		case Raise:
			err = poker.Bet(g, pn, a.Amount)
		case Call:
			err = poker.Bet(g, pn, min(top-p.Bet, p.Stack))
		case Check:
			err = poker.Bet(g, pn, 0)
		default:
			err = poker.Fold(g, pn, 0)
		}
		if err != nil {
			t.Fatalf("%s's %+v rejected: %v", name(pn), a, err)
		}
	}
}

// totals fills in each seat's chips won or lost.
func totals(g *poker.Game, res []stats) []stats {
	for pn, p := range g.GenerateOmniView().Players {
		res[pn].net = int(p.Stack+p.TotalBet) - int(p.TotalBuyIn)
	}
	return res
}

// startHand resets flags for the hand just dealt and logs who has what.
func startHand(view *poker.GameView, n int, flags []handFlags, log io.Writer, name func(uint) string) {
	fmt.Fprintf(log, "\nhand %d, %s deals\n", n, name(view.DealerNum))
	for pn, p := range view.Players {
		flags[pn] = handFlags{}
		if p.In {
			flags[pn].start = p.Stack + p.TotalBet
			fmt.Fprintf(log, "  %-8s %5d  %s\n", name(uint(pn)), flags[pn].start, cardList(p.Cards[:]))
		}
	}
}

// finishHand adds the hand that just ended to res and logs how it went.
func finishHand(g *poker.Game, view *poker.GameView, res []stats, flags []handFlags, log io.Writer, name func(uint) string) {
	won := map[uint]bool{}
	for _, pot := range view.Pots {
		if pot.Amt == 0 {
			continue
		}
		var names []string
		for _, pn := range pot.WinningPlayerNums {
			won[pn] = true
			names = append(names, name(pn))
		}
		fmt.Fprintf(log, "  %s wins %d\n", strings.Join(names, ", "), pot.Amt)
	}
	for _, pn := range g.ShownPlayerNums() {
		res[pn].showdowns++
		if won[pn] {
			res[pn].wonSD++
		}
		fmt.Fprintf(log, "  %s shows %s\n", name(pn), cardList(view.Players[pn].Cards[:]))
	}
	for pn, f := range flags {
		if f.start == 0 {
			continue // not dealt in
		}
		s := &res[pn]
		s.hands++
		s.vpip += b2i(f.vpip)
		s.pfr += b2i(f.pfr)
		s.reraiseSpots += b2i(f.reraiseSpot)
		s.bets3 += b2i(f.reraised)
		s.sawFlop += b2i(len(dealt(view.CommunityCards)) >= 3 && !f.foldedPreflop)
		if net := int(view.Players[pn].Stack) - int(f.start); net != 0 {
			fmt.Fprintf(log, "  %-8s %+d\n", name(uint(pn)), net)
		}
	}
}

// note counts pn's move a. view is from before it.
func note(s *stats, f *handFlags, view *poker.GameView, pn uint, a Action) {
	p := view.Players[pn]
	var top, pot uint
	for _, q := range view.Players {
		top = max(top, q.Bet)
		pot += q.TotalBet
	}
	owe := top - p.Bet
	if view.Stage == poker.PreFlop {
		raised := top > view.Config.BigBlind
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
		s.sizes[sizeOf(a.Amount, owe, pot, p.Stack)]++
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
	t.Logf("%-8s %6s %5s %5s %5s %5s %5s %5s %5s %7s   postflop sizes %v",
		"seat", "hands", "vpip", "pfr", "3bet", "af", "fold", "wtsd", "w$sd", "bb/100", sizeNames)
	for pn, s := range res {
		sizes := make([]string, len(s.sizes))
		for i, n := range s.sizes {
			sizes[i] = pct(n, s.aggressive)
		}
		t.Logf("%-8s %6d %5s %5s %5s %5.1f %5s %5s %5s %+7.1f   %s",
			fmt.Sprintf("%s%d", s.level, pn+1), s.hands,
			pct(s.vpip, s.hands), pct(s.pfr, s.hands), pct(s.bets3, s.reraiseSpots),
			float64(s.aggressive)/float64(max(s.calls, 1)),
			pct(s.foldBet, s.facedBet), pct(s.showdowns, s.sawFlop), pct(s.wonSD, s.showdowns),
			float64(s.net)/simBB/float64(max(s.hands, 1))*100, strings.Join(sizes, " "))
	}
	t.Logf("%-8s %6s %5s %5s %5s %5s %5s %5s %5s", "human", "", "22-28", "17-22", "6-9", "2-3", "40-55", "25-30", "50-55")
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

func streetName(view *poker.GameView) string {
	switch view.Stage {
	case poker.PreFlop:
		return "preflop"
	case poker.Flop:
		return "flop"
	case poker.Turn:
		return "turn"
	}
	return "river"
}

func cardList(cards []eval.Card) string {
	var names []string
	for _, c := range dealt(cards) {
		names = append(names, c.String())
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
