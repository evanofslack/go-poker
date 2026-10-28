package holdem

import "testing"

// potGame is a pot limit game with blinds 10/20 and the stacks given.
func potGame(t *testing.T, stacks ...uint) *State {
	t.Helper()
	s := game(t, stacks...)
	s.Config.Game = Game{Hole: 4, Pot: true}
	return s
}

func TestPotLimit(t *testing.T) {
	s := potGame(t, 5000, 5000, 5000)
	start(t, s)
	// 30 in, 20 to call: the pot after calling is 70
	if l := s.Legal(s.ToAct); l.RaiseMin != 40 || l.RaiseMax != 70 {
		t.Fatalf("under the gun %+v, want 40 to 70", l)
	}
	if _, err := s.Act(s.ToAct, Raise{To: 71}); err == nil {
		t.Fatal("raised over the pot")
	}
	act(t, s, s.ToAct, Raise{To: 70})
	// the small blind owes 60 into 100: 70 + 160
	if l := s.Legal(s.ToAct); l.RaiseMax != 230 {
		t.Fatalf("small blind %+v, want a pot raise to 230", l)
	}
	act(t, s, s.ToAct, Call{})
	act(t, s, s.ToAct, Call{})
	street(t, s)
	// a bet on the flop is up to the pot
	if l := s.Legal(s.ToAct); l.RaiseMin != 20 || l.RaiseMax != 210 {
		t.Fatalf("flop %+v, want 20 to 210", l)
	}
	act(t, s, s.ToAct, Raise{To: 100})
	// 310 in, 100 to call: 100 + 410
	if l := s.Legal(s.ToAct); l.RaiseMax != 510 {
		t.Fatalf("facing a bet %+v, want 510", l)
	}
}

func TestPotLimitAntes(t *testing.T) {
	s := potGame(t, 5000, 5000, 5000)
	s.Config.Ante = 5
	start(t, s)
	// the antes are in the pot: 45 in, 20 to call
	if l := s.Legal(s.ToAct); l.RaiseMax != 85 {
		t.Fatalf("%+v, want 85", l)
	}
}

func TestPotLimitShortStack(t *testing.T) {
	s := potGame(t, 50, 5000, 5000)
	start(t, s)
	// under the gun can only go all in, below the pot
	if l := s.Legal(s.ToAct); l.RaiseMax != 50 {
		t.Fatalf("%+v, want all in for 50", l)
	}
	act(t, s, s.ToAct, Raise{To: 50})
	// the blinds can raise over it: 80 in, 40 to call, 50 + 120
	if l := s.Legal(s.ToAct); l.RaiseMin != 80 || l.RaiseMax != 170 {
		t.Fatalf("small blind %+v", l)
	}
}

func TestNoLimitUncapped(t *testing.T) {
	s := game(t, 5000, 5000, 5000)
	s.Config.Game = Game{Hole: 4}
	start(t, s)
	if l := s.Legal(s.ToAct); l.RaiseMax != 5000 {
		t.Fatalf("no limit %+v", l)
	}
}
