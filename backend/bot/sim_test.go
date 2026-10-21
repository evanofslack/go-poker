package bot

import (
	"math/rand/v2"
	"os"
	"testing"

	"github.com/evanofslack/go-poker/poker"
)

// TestSim plays medium against easy heads up on the real engine and checks
// medium comes out ahead. Shuffles aren't seeded and it takes a while, so it
// only runs with BOTSIM=1.
func TestSim(t *testing.T) {
	if os.Getenv("BOTSIM") == "" {
		t.Skip("set BOTSIM=1 to run")
	}
	const hands, buyIn = 2000, 2000
	levels := []string{Medium, Easy}
	net := sim(t, levels, hands, buyIn)
	for i, level := range levels {
		t.Logf("%s: %+.1f big blinds per 100 hands", level, float64(net[i])/20/hands*100)
	}
	if net[0] <= 0 {
		t.Errorf("medium lost %d chips to easy over %d hands", -net[0], hands)
	}
}

// sim plays hands between bots of levels, rebuying busted ones, and returns
// each one's chips won or lost.
func sim(t *testing.T, levels []string, hands int, buyIn uint) []int {
	g := poker.NewGame(poker.GameConfig{MaxBuy: buyIn, BigBlind: 20, SmallBlind: 10})
	for i := range levels {
		pn := g.AddPlayer()
		must(t, poker.BuyIn(g, pn, buyIn))
		must(t, poker.ToggleReady(g, pn, 0))
		must(t, poker.SetSeatID(g, pn, uint(i+1)))
	}
	// players stay in seat order, so the index is the level's index
	must(t, g.Start())
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	for played := 0; played < hands; {
		view := g.GenerateOmniView()
		if !view.Betting {
			played++
			for pn, p := range view.Players {
				if p.Stack == 0 {
					must(t, poker.BuyIn(g, uint(pn), buyIn))
					if !p.Ready {
						must(t, poker.ToggleReady(g, uint(pn), 0))
					}
				}
			}
			must(t, poker.Deal(g, view.DealerNum, 0))
			continue
		}
		pn := view.ActionNum
		p := view.Players[pn]
		a := Decide(levels[pn], g.GeneratePlayerView(pn), pn, r)
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
			t.Fatalf("%s's %+v rejected: %v", levels[pn], a, err)
		}
	}

	net := make([]int, len(levels))
	for pn, p := range g.GenerateOmniView().Players {
		net[pn] = int(p.Stack+p.TotalBet) - int(p.TotalBuyIn)
	}
	return net
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
