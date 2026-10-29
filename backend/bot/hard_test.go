package bot

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// hardSpot deals a six handed hand with stack chips each, plays line's moves
// by the players to act, and returns the spot of the player to act then,
// holding hole.
func hardSpot(t *testing.T, stack uint, hole []eval.Card, moves ...holdem.Action) spot {
	t.Helper()
	g := holdem.New(holdem.Config{SmallBlind: 10, BigBlind: 20}, [32]byte{3})
	for seat := 1; seat <= 6; seat++ {
		must(t, g.Sit(seat, stack))
	}
	events, err := g.StartHand(holdem.Deal{})
	must(t, err)
	var line []holdem.Acted
	add := func(events []holdem.Event) {
		for _, e := range events {
			if a, ok := e.(holdem.Acted); ok {
				line = append(line, a)
			}
		}
	}
	add(events)
	for _, m := range moves {
		events, err := g.Act(g.ToAct, m)
		must(t, err)
		add(events)
	}
	s := spotOf(g.View(g.ToAct), g.ToAct, line)
	s.hole = hole
	return s
}

func TestHardPreflop(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 20 {
		if a := hard(hardSpot(t, 2000, hole("7s", "2d")), r); a.Kind != Fold {
			t.Fatalf("seven deuce under the gun: %+v", a)
		}
		if a := hard(hardSpot(t, 2000, hole("As", "Ad")), r); a.Kind != Raise {
			t.Fatalf("aces under the gun: %+v", a)
		}
		// under the gun raises, and the next player has kings
		if a := hard(hardSpot(t, 2000, hole("Ks", "Kd"), holdem.Raise{To: 60}), r); a.Kind != Raise {
			t.Fatalf("kings facing an open: %+v", a)
		}
		// three players fold to the small blind, ten big blinds deep
		s := hardSpot(t, 200, hole("As", "Td"), holdem.Fold{}, holdem.Fold{}, holdem.Fold{})
		if a := hard(s, r); a.Kind != Raise || a.Amount != s.stack {
			t.Fatalf("ace ten pushes short from the small blind: %+v", a)
		}
	}
}

func TestHardFoldsToStrength(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	// under the gun opens, the next player reraises, then four-bets and gets
	// five-bet: queen jack folds, aces don't
	moves := []holdem.Action{holdem.Raise{To: 60}, holdem.Raise{To: 200}, holdem.Fold{}, holdem.Fold{}, holdem.Fold{}, holdem.Fold{}, holdem.Raise{To: 500}}
	for range 10 {
		if a := hard(hardSpot(t, 4000, hole("Qs", "Jd"), moves...), r); a.Kind != Fold {
			t.Fatalf("queen jack facing a four-bet: %+v", a)
		}
		if a := hard(hardSpot(t, 4000, hole("As", "Ad"), moves...), r); a.Kind == Fold {
			t.Fatalf("aces facing a four-bet: %+v", a)
		}
	}
}

func TestWeighNarrows(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	// under the gun opens and is three-bet: the opener's range is far
	// stronger than a random hand
	s := hardSpot(t, 2000, hole("2s", "3d"), holdem.Raise{To: 60}, holdem.Raise{To: 200})
	p := newPool(s.game, s.hole, [][]eval.Card{nil}, r)
	ws := weigh(s, p)
	if len(ws) != 5 {
		t.Fatalf("weights for %d opponents, want 5", len(ws))
	}
	for seat, w := range ws {
		var sum, pct float64
		for i, x := range w {
			sum += x
			pct += x * p.pre[i]
		}
		mean := pct / sum
		switch {
		case seat == s.line[3].Seat && mean < 0.9:
			t.Errorf("the three-bettor's hands average %.2f", mean)
		case seat == s.line[2].Seat && mean < 0.8:
			t.Errorf("the opener's hands average %.2f", mean)
		case seat != s.line[2].Seat && seat != s.line[3].Seat && math.Abs(mean-0.5) > 0.05:
			t.Errorf("seat %d hasn't acted, but its hands average %.2f", seat, mean)
		}
	}
}

func TestContOf(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	p := newPool(holdem.Game{}, hole("2s", "3d"), [][]eval.Card{cards("Ks", "8h", "4c")}, r)
	made, draws := p.strength(3)
	w := make([]float64, len(made))
	for i := range w {
		w[i] = 1
	}
	t1 := tendencies{fold: 0.5, aggro: 1}
	call, raise := contOf(made, draws, w, 0.6, false, t1)
	if c := share(w, call) + share(w, raise); math.Abs(c-0.5) > 0.06 {
		t.Errorf("a player who folds half the time continues %.2f against a 60%% bet", c)
	}
	big, _ := contOf(made, draws, w, 2, false, t1)
	if share(w, big) >= share(w, call) {
		t.Error("more continue against an overbet")
	}
	sticky, _ := contOf(made, draws, w, 0.6, false, tendencies{fold: 0.2, aggro: 1})
	if share(w, sticky) <= share(w, call) {
		t.Error("a calling station continues no more")
	}
}

func TestStrengthDoubleBoard(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	p := newPool(holdem.Game{}, hole("2s", "3d"), [][]eval.Card{cards("Ks", "8h", "4c"), cards("Qd", "Jd", "9s")}, r)
	made, _ := p.strength(3)
	lo, hi, sum := 1.0, 0.0, 0.0
	for _, m := range made {
		lo, hi, sum = min(lo, m), max(hi, m), sum+m
	}
	if lo > 0.01 || hi < 0.99 || math.Abs(sum/float64(len(made))-0.5) > 0.01 {
		t.Errorf("double board strengths run %.2f to %.2f, average %.2f", lo, hi, sum/float64(len(made)))
	}
}

func TestPercentileOmaha(t *testing.T) {
	g := holdem.Game{Hole: 4, Pot: true}
	good, bad := percentile(g, cards("As", "Ad", "Ks", "Kd")), percentile(g, cards("9s", "4d", "Kc", "2h"))
	if good < 0.99 || bad > 0.1 {
		t.Errorf("aces kings double suited %.3f, junk %.3f", good, bad)
	}
}
