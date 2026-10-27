package holdem

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// seated is a game of n players with stack chips each.
func seated(t *testing.T, n int, stack uint) *State {
	t.Helper()
	return game(t, slices.Repeat([]uint{stack}, n)...)
}

func startStraddles(t *testing.T, s *State, n int) []Event {
	t.Helper()
	events, err := s.StartHand(n)
	return must(t, events, err)
}

func TestPositions(t *testing.T) {
	for _, tc := range []struct {
		players  int
		straddle []int
	}{
		{2, nil},
		{3, nil},
		{4, []int{4}},
		{5, []int{4, 5}},
		{9, []int{4, 5, 6, 7, 8, 9}},
	} {
		s := seated(t, tc.players, 10000)
		before := s.Clone()
		pos := s.Positions()
		if !reflect.DeepEqual(*s, before) {
			t.Fatalf("%d handed: Positions changed the state", tc.players)
		}
		if !slices.Equal(pos.Straddle, tc.straddle) {
			t.Fatalf("%d handed: straddle %v, want %v", tc.players, pos.Straddle, tc.straddle)
		}
		startStraddles(t, s, len(pos.Straddle))
		if s.Button != pos.Button || s.SB != pos.SB || s.BB != pos.BB || !slices.Equal(s.Straddles, pos.Straddle) {
			t.Fatalf("%d handed: positions %+v, hand button %d sb %d bb %d straddles %v",
				tc.players, pos, s.Button, s.SB, s.BB, s.Straddles)
		}
	}
	if pos := game(t, 1000).Positions(); !reflect.DeepEqual(pos, Positions{}) {
		t.Fatalf("one player: %+v, want zero", pos)
	}
}

func TestStraddleNeedsChipsBehind(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ante     uint
		stacks   []uint
		straddle []int
	}{
		{"exactly the straddle", 0, []uint{1000, 1000, 1000, 40, 1000}, nil},
		{"a chip behind", 0, []uint{1000, 1000, 1000, 41, 1000}, []int{4, 5}},
		{"exactly after the ante", 5, []uint{1000, 1000, 1000, 45, 1000}, nil},
		{"a chip behind after the ante", 5, []uint{1000, 1000, 1000, 46, 1000}, []int{4, 5}},
		{"re-straddle short", 0, []uint{1000, 1000, 1000, 1000, 80, 1000}, []int{4}},
	} {
		s := game(t, tc.stacks...)
		s.Config.Ante = tc.ante
		if got := s.Positions().Straddle; !slices.Equal(got, tc.straddle) {
			t.Errorf("%s: straddle %v, want %v", tc.name, got, tc.straddle)
		}
	}
}

func TestStraddleAmountsAndOrder(t *testing.T) {
	for n := 1; n <= 3; n++ {
		s := seated(t, n+3, 1000)
		events := startStraddles(t, s, n)
		var posted []uint
		for _, e := range events {
			if a, ok := e.(Acted); ok && a.Kind == KindStraddle {
				posted = append(posted, a.To)
			}
		}
		want := []uint{40, 80, 160}[:n]
		if !slices.Equal(posted, want) || len(s.Straddles) != n {
			t.Fatalf("%d straddles: posted %v, want %v", n, posted, want)
		}
		// the button acts first, and a raise is at least the last straddle more
		if s.ToAct != 1 {
			t.Fatalf("%d straddles: to act %d, want 1", n, s.ToAct)
		}
		if l := s.Legal(1); l.Call != want[n-1] || l.RaiseMin != 2*want[n-1] {
			t.Fatalf("%d straddles: legal %+v", n, l)
		}
		// then round through the blinds to the straddlers
		order := []int{1}
		for s.Betting && len(order) < 2*(n+3) {
			var a Action = Call{}
			if s.Legal(s.ToAct).Check {
				a = Check{}
			}
			act(t, s, s.ToAct, a)
			order = append(order, s.ToAct)
		}
		wantOrder := []int{1, 2, 3, 4, 5, 6}[:n+3]
		if !slices.Equal(order[:n+3], wantOrder) {
			t.Fatalf("%d straddles: order %v, want %v", n, order, wantOrder)
		}
	}
}

func TestStraddleOption(t *testing.T) {
	s := seated(t, 5, 1000)
	startStraddles(t, s, 2)
	act(t, s, 1, Call{})
	act(t, s, 2, Call{})
	act(t, s, 3, Call{})
	// the first straddler faces the re-straddle
	if l := s.Legal(4); l.Call != 40 || l.Check {
		t.Fatalf("first straddler %+v, want to call 40", l)
	}
	act(t, s, 4, Call{})
	if l := s.Legal(5); !l.Check || l.RaiseMin != 160 || l.RaiseMax != 1000 {
		t.Fatalf("straddler's option %+v", l)
	}
	act(t, s, 5, Check{})
	if s.Betting || s.Street != Preflop {
		t.Fatal("preflop should be closed after the straddler checks")
	}
	street(t, s)
	if s.ToAct != 2 || s.MinRaise != 20 {
		t.Fatalf("flop: to act %d min raise %d, want 2 and 20", s.ToAct, s.MinRaise)
	}
}

func TestStraddleRaisedReopens(t *testing.T) {
	s := seated(t, 4, 1000)
	startStraddles(t, s, 1)
	act(t, s, 1, Raise{To: 80})
	act(t, s, 2, Fold{})
	act(t, s, 3, Fold{})
	if l := s.Legal(4); l.Call != 40 || l.RaiseMin != 120 {
		t.Fatalf("straddler facing a raise %+v", l)
	}
}

func TestFoldToStraddler(t *testing.T) {
	s := seated(t, 4, 1000)
	startStraddles(t, s, 1)
	act(t, s, 1, Fold{})
	act(t, s, 2, Fold{})
	act(t, s, 3, Fold{})
	if s.InHand() {
		t.Fatal("hand should be over")
	}
	if got := stacks(s); !slices.Equal(got, []uint{1000, 990, 980, 1030}) {
		t.Fatalf("stacks %v", got)
	}
}

func TestStraddleWithAntes(t *testing.T) {
	s := seated(t, 5, 1000)
	s.Config.Ante = 5
	startStraddles(t, s, 2)
	if got := stacks(s); !slices.Equal(got, []uint{995, 985, 975, 955, 915}) {
		t.Fatalf("stacks %v", got)
	}
	var potted uint
	for _, p := range s.Pots {
		potted += p.Amount
	}
	if potted != 5*5+10+20+40+80 {
		t.Fatalf("pots %d", potted)
	}
	if v := s.View(0); !slices.Equal(v.Straddles, []int{4, 5}) {
		t.Fatalf("view straddles %v", v.Straddles)
	}
}

func TestStraddleIllegal(t *testing.T) {
	s := seated(t, 4, 1000)
	before := s.Clone()
	for _, n := range []int{-1, 2} {
		if _, err := s.StartHand(n); !errors.Is(err, ErrIllegal) {
			t.Fatalf("StartHand(%d): %v, want ErrIllegal", n, err)
		}
		if !reflect.DeepEqual(*s, before) {
			t.Fatalf("StartHand(%d) changed the state", n)
		}
	}
	// straddles clear in the next hand
	startStraddles(t, s, 1)
	act(t, s, 1, Fold{})
	act(t, s, 2, Fold{})
	act(t, s, 3, Fold{})
	start(t, s)
	if len(s.Straddles) != 0 {
		t.Fatalf("straddles %v carried over", s.Straddles)
	}
}
