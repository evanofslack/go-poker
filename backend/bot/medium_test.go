package bot

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

func TestChen(t *testing.T) {
	tests := []struct {
		hole []eval.Card
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
	near("aces heads up", equity(holdem.Game{}, hole("As", "Ad"), nil, 1, anyHand, 0, 4000, r), 0.85)
	near("seven deuce heads up", equity(holdem.Game{}, hole("7s", "2d"), nil, 1, anyHand, 0, 4000, r), 0.35)
	if e := equity(holdem.Game{}, hole("As", "Ks"), cards("Qs", "Js", "Ts", "2d", "3c"), 3, anyHand, 0, 500, r); e != 1 {
		t.Errorf("royal flush: equity = %v, want 1", e)
	}
	one := equity(holdem.Game{}, hole("Qs", "Qd"), nil, 1, anyHand, 0, 2000, r)
	five := equity(holdem.Game{}, hole("Qs", "Qd"), nil, 5, anyHand, 0, 2000, r)
	if five >= one {
		t.Errorf("queens: equity against 5 (%.2f) should be below against 1 (%.2f)", five, one)
	}
	if e := equity(holdem.Game{}, hole("7s", "2d"), nil, 0, anyHand, 0, 100, r); e != 1 {
		t.Errorf("no opponents: equity = %v, want 1", e)
	}
}

func BenchmarkEquity(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	for b.Loop() {
		equity(holdem.Game{}, hole("As", "Kd"), nil, 8, anyHand, 0, trials, r)
	}
}

// at is player to act holding h on board, owing owe into pot, against others.
func at(h []eval.Card, board []eval.Card, owe, pot uint, others int) spot {
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
			hole:     []eval.Card{deck[0], deck[1]},
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

// unopened is a six-handed preflop spot nobody has raised, behind players
// from the button.
func unopened(h []eval.Card, behind int) spot {
	return spot{hole: h, top: 20, stack: 2000, minRaise: 20, pot: 30, bb: 20, others: 5, behind: behind}
}

func TestMediumOpensOrFolds(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		for behind := range 5 {
			if a := medium(unopened(hole("As", "Ad"), behind), r); a.Kind != Raise {
				t.Fatalf("aces unopened with %d behind: got %v, want raise", behind, a)
			}
			if a := medium(unopened(hole("7s", "2d"), behind), r); a.Kind != Fold {
				t.Fatalf("seven deuce unopened with %d behind: got %v, want fold", behind, a)
			}
		}
		if a := medium(unopened(hole("9s", "8s"), 3), r); a.Kind == Call {
			t.Fatalf("limped nine eight suited: %v", a)
		}
	}
}

func TestMediumOpensWiderOnTheButton(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	if a := medium(unopened(hole("6s", "5s"), 0), r); a.Kind != Raise {
		t.Fatalf("six five suited on the button: got %v, want raise", a)
	}
	if a := medium(unopened(hole("6s", "5s"), 4), r); a.Kind != Fold {
		t.Fatalf("six five suited first to act: got %v, want fold", a)
	}
}

func TestMediumReraisesAces(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := unopened(hole("As", "Ad"), 2)
	s.top, s.pot = 60, 90
	for range 200 {
		a := medium(s, r)
		if a.Kind != Raise || a.Amount < 3*60 {
			t.Fatalf("aces facing a raise to 60: got %+v, want a 3-bet", a)
		}
	}
}

func TestDrawOf(t *testing.T) {
	tests := []struct {
		hole  []eval.Card
		board []eval.Card
		want  draw
	}{
		{hole("As", "Ks"), cards("Qs", "7s", "2d"), openEnded},    // flush draw
		{hole("9h", "8d"), cards("7c", "6s", "2d"), openEnded},    // five to ten
		{hole("9h", "8d"), cards("6c", "5s", "Kd"), gutshot},      // needs a seven
		{hole("Ah", "2d"), cards("3c", "4s", "Kd"), gutshot},      // wheel needs a five
		{hole("Js", "Ts"), cards("9s", "8d", "2s"), comboDraw},    // flush and straight
		{hole("9h", "8d"), cards("7c", "6s", "5d"), noDraw},       // made straight
		{hole("Ah", "Kd"), cards("9c", "8s", "7d", "6h"), noDraw}, // board's own draw
		{hole("As", "Ks"), cards("Qs", "7s", "2d", "3c", "4h"), noDraw},
	}
	for _, tt := range tests {
		if got := drawOf(tt.hole, tt.board); got != tt.want {
			t.Errorf("drawOf(%v, %v) = %v, want %v", tt.hole, tt.board, got, tt.want)
		}
	}
}

func TestWet(t *testing.T) {
	for _, b := range [][]eval.Card{cards("Ks", "7s", "2d"), cards("9c", "8d", "6h"), cards("Ah", "2c", "4d")} {
		if !wet(b) {
			t.Errorf("wet(%v) = false", b)
		}
	}
	for _, b := range [][]eval.Card{cards("Ks", "7d", "2c"), cards("Ah", "Ac", "8d")} {
		if wet(b) {
			t.Errorf("wet(%v) = true", b)
		}
	}
}

// TestMediumCbets checks the preflop raiser bets the flop often, but not with
// every hand.
func TestMediumCbets(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := at(hole("7s", "2d"), cards("Kh", "9c", "4d"), 0, 120, 1)
	s.led, s.raised, s.behind = true, true, 0
	bets := 0
	for range 400 {
		if medium(s, r).Kind == Raise {
			bets++
		}
	}
	if bets < 150 || bets > 350 {
		t.Errorf("c-bet seven deuce on a dry flop %d times in 400, want about 260", bets)
	}
}

func TestStyleOf(t *testing.T) {
	a, b := styleOf("a"), styleOf("b")
	if a != styleOf("a") {
		t.Error("styleOf changed for the same uuid")
	}
	if a == b {
		t.Error("styleOf gave two uuids the same style")
	}
	for _, s := range []style{a, b, styleOf("")} {
		if s.loose < -1 || s.loose > 1.5 || s.aggro < -0.3 || s.aggro > 0.3 {
			t.Errorf("style %+v out of range", s)
		}
	}
}
