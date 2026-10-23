package bot

import (
	"math/rand/v2"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

func cards(s ...string) []eval.Card {
	var cs []eval.Card
	for _, c := range s {
		cs = append(cs, eval.MustParseCardString(c))
	}
	return cs
}

func hole(a, b string) [2]eval.Card {
	c := cards(a, b)
	return [2]eval.Card{c[0], c[1]}
}

func TestStrength(t *testing.T) {
	tests := []struct {
		name  string
		hole  [2]eval.Card
		board []eval.Card
		want  strength
	}{
		{"hidden cards", [2]eval.Card{}, nil, weak},
		{"aces", hole("As", "Ad"), nil, strong},
		{"low pair", hole("2s", "2d"), nil, fair},
		{"jack queen", hole("Js", "Qd"), nil, fair},
		{"suited ace", hole("As", "4s"), nil, fair},
		{"junk", hole("7s", "2d"), nil, weak},
		{"high card", hole("7s", "2d"), cards("Ks", "9h", "4c"), weak},
		{"pair", hole("7s", "2d"), cards("7h", "9h", "4c"), fair},
		{"two pair on the turn", hole("7s", "2d"), cards("7h", "2h", "4c", "Kd"), strong},
		{"flush on the river", hole("As", "3s"), cards("7s", "9s", "Ks", "2d", "2c"), strong},
	}
	for _, tt := range tests {
		if got := strengthOf(tt.hole, tt.board); got != tt.want {
			t.Errorf("%s: strength = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// facing is player to act holding h, facing a bet of owe.
func facing(h [2]eval.Card, owe, stack uint) spot {
	return spot{hole: h, top: owe, stack: stack, minRaise: 20}
}

func TestEasyNeverFoldsForFree(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		a := easy(facing(hole("7s", "2d"), 0, 1000), r)
		if a.Kind != Check {
			t.Fatalf("junk with nothing to call: got %v, want check", a)
		}
	}
}

func TestEasyFoldsJunkToBigBets(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		a := easy(facing(hole("7s", "2d"), 500, 1000), r)
		if a.Kind != Fold {
			t.Fatalf("junk facing half its stack: got %v, want fold", a)
		}
	}
}

func TestEasyRaisesLegally(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	raised := false
	for range 1000 {
		a := easy(facing(hole("As", "Ad"), 20, 1000), r)
		switch a.Kind {
		case Raise:
			raised = true
			// a min-raise: call 20 and raise by MinRaise
			if a.Amount != 40 {
				t.Fatalf("raise amount = %d, want 40", a.Amount)
			}
		case Fold:
			t.Fatal("folded aces")
		}
	}
	if !raised {
		t.Fatal("never raised aces")
	}
}

func TestEasyCallsShortStack(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		// the min-raise is more than its stack, so it can only call or fold
		a := easy(facing(hole("As", "Ad"), 20, 30), r)
		if a.Kind != Call {
			t.Fatalf("aces with a short stack: got %v, want call", a)
		}
	}
}

func TestSpotOf(t *testing.T) {
	g := holdem.New(holdem.Config{SmallBlind: 10, BigBlind: 20}, [32]byte{})
	for seat := 1; seat <= 3; seat++ {
		must(t, g.Sit(seat, 1000))
	}
	_, err := g.StartHand()
	must(t, err)
	s := spotOf(g.View(g.ToAct), g.ToAct)
	if len(s.board) != 0 || s.top != 20 || s.pot != 30 || s.others != 2 || s.bb != 20 || s.hole[0] == 0 {
		t.Fatalf("preflop spot %+v", s)
	}
}
