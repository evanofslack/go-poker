package bot

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
)

func TestChen(t *testing.T) {
	tests := []struct {
		hole [2]eval.Card
		want float64
	}{
		{hole("As", "Ad"), 20},
		{hole("Ks", "Kd"), 16},
		{hole("2s", "2d"), 5},
		{hole("As", "Ks"), 12},
		{hole("7s", "2d"), -1},
		{hole("Ts", "9s"), 8}, // 5, suited +2, connected and small +1
		{hole("5d", "7d"), 6}, // 3.5, suited +2, one gap -1, small +1, rounded up
	}
	for _, tt := range tests {
		if got := chen(tt.hole); got != tt.want {
			t.Errorf("chen(%v) = %v, want %v", tt.hole, got, tt.want)
		}
	}
}

func TestEquity(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	near := func(name string, got, want float64) {
		if math.Abs(got-want) > 0.04 {
			t.Errorf("%s: equity = %.3f, want about %.2f", name, got, want)
		}
	}
	near("aces heads up", equity(hole("As", "Ad"), nil, 1, 4000, r), 0.85)
	near("seven deuce heads up", equity(hole("7s", "2d"), nil, 1, 4000, r), 0.35)
	if e := equity(hole("As", "Ks"), cards("Qs", "Js", "Ts", "2d", "3c"), 3, 500, r); e != 1 {
		t.Errorf("royal flush: equity = %v, want 1", e)
	}
	one := equity(hole("Qs", "Qd"), nil, 1, 2000, r)
	five := equity(hole("Qs", "Qd"), nil, 5, 2000, r)
	if five >= one {
		t.Errorf("queens: equity against 5 (%.2f) should be below against 1 (%.2f)", five, one)
	}
	if e := equity(hole("7s", "2d"), nil, 0, 100, r); e != 1 {
		t.Errorf("no opponents: equity = %v, want 1", e)
	}
}

func BenchmarkEquity(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	for b.Loop() {
		equity(hole("As", "Kd"), nil, 8, trials, r)
	}
}

// at is player to act holding h on board, owing owe into pot, against others.
func at(h [2]eval.Card, board []eval.Card, owe, pot uint, others int) spot {
	return spot{hole: h, board: board, top: owe, stack: 2000, minRaise: 20, pot: pot, bb: 20, others: others}
}

func TestMediumNeverFoldsForFree(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		for _, s := range []spot{
			at(hole("7s", "2d"), nil, 0, 40, 1),
			at(hole("7s", "2d"), cards("Ks", "9h", "4c"), 0, 40, 3),
		} {
			if a := medium(s, r); a.Kind == Fold {
				t.Fatalf("folded %v with nothing to call", s.hole)
			}
		}
	}
}

func TestMediumFoldsJunkPreflop(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		if a := medium(at(hole("7s", "2d"), nil, 60, 90, 3), r); a.Kind != Fold {
			t.Fatalf("seven deuce facing a raise: got %v, want fold", a)
		}
	}
}

func TestMediumCallsWithTheNuts(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		a := medium(at(hole("As", "Ks"), cards("Qs", "Js", "Ts", "2d", "3c"), 50, 300, 1), r)
		if a.Kind != Call && a.Kind != Raise {
			t.Fatalf("royal flush facing a bet: got %v", a)
		}
	}
}

// TestMediumMovesAreLegal checks every bet or raise is at least a min-raise
// or all in, and never more than the stack, over many random spots.
func TestMediumMovesAreLegal(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	deck := append([]eval.Card{}, eval.DefaultDeck...)
	for range 500 {
		r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
		s := spot{
			hole:     [2]eval.Card{deck[0], deck[1]},
			board:    append([]eval.Card{}, deck[2:2+[]int{0, 3, 4, 5}[r.IntN(4)]]...),
			stack:    uint(r.IntN(3000)) + 1,
			minRaise: 20,
			bb:       20,
			others:   r.IntN(5) + 1,
			called:   r.IntN(4) == 0,
		}
		s.bet = uint(r.IntN(200))
		s.top = s.bet + uint(r.IntN(400))
		s.pot = s.top + uint(r.IntN(1000))

		a := medium(s, r)
		owe := s.top - s.bet
		switch a.Kind {
		case Raise:
			if s.called {
				t.Fatalf("raised after acting: %+v", s)
			}
			if a.Amount > s.stack || (a.Amount < owe+s.minRaise && a.Amount != s.stack) {
				t.Fatalf("raise of %d isn't legal: owe %d, min raise %d, stack %d", a.Amount, owe, s.minRaise, s.stack)
			}
		case Check:
			if owe > 0 {
				t.Fatalf("checked owing %d", owe)
			}
		}
	}
}
