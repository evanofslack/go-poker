package bot

import (
	"math/rand/v2"
	"os"
	"testing"

	"github.com/evanofslack/go-poker/holdem"
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
	g := holdem.New(holdem.Config{SmallBlind: 10, BigBlind: 20}, [32]byte{byte(rand.Uint64())})
	bought := make([]uint, len(levels))
	for i := range levels {
		must(t, g.Sit(i+1, buyIn))
		bought[i] = buyIn
	}
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	for played := 0; played < hands; {
		switch {
		case !g.InHand():
			played++
			for i, p := range g.Players {
				if p.Stack == 0 {
					must(t, g.AddChips(p.Seat, buyIn))
					must(t, g.SetSittingOut(p.Seat, false))
					bought[i] += buyIn
				}
			}
			_, err := g.StartHand()
			must(t, err)
		case !g.Betting:
			_, err := g.NextStreet()
			must(t, err)
		default:
			seat := g.ToAct
			a := Decide(levels[seat-1], g.View(seat), seat, r)
			if _, err := g.Act(seat, a); err != nil {
				t.Fatalf("%s's %+v rejected: %v", levels[seat-1], a, err)
			}
		}
	}

	net := make([]int, len(levels))
	for i, p := range g.Players {
		net[i] = int(p.Stack+p.Committed) - int(bought[i])
	}
	return net
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
