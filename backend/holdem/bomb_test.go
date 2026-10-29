package holdem

import (
	"errors"
	"testing"
)

func bombHand(t *testing.T, s *State, bomb uint) []Event {
	t.Helper()
	events, err := s.StartHand(Deal{Bomb: bomb})
	return must(t, events, err)
}

func TestBomb(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	events := bombHand(t, s, 50)
	var bombs int
	for _, e := range events {
		if a, ok := e.(Acted); ok {
			if a.Kind != KindBomb || a.Amount != 50 || a.Street != Preflop {
				t.Fatalf("posted %+v", a)
			}
			bombs++
		}
	}
	if bombs != 3 {
		t.Fatalf("%d bombs posted", bombs)
	}
	if s.Street != Flop || len(s.Board) != 3 || !s.Betting || s.Bomb != 50 {
		t.Fatalf("street %q board %v betting %v", s.Street, s.Board, s.Betting)
	}
	// left of the button acts first, and can check or bet a big blind
	if s.ToAct != 2 {
		t.Fatalf("seat %d to act, want 2", s.ToAct)
	}
	if l := s.Legal(2); !l.Check || l.RaiseMin != 20 {
		t.Fatalf("first to act %+v", l)
	}
	if len(s.Pots) != 1 || s.Pots[0].Amount != 150 {
		t.Fatalf("pots %+v", s.Pots)
	}
	act(t, s, 2, Check{})
	act(t, s, 3, Check{})
	act(t, s, 1, Check{})
	if s.Street != Flop || s.Betting {
		t.Fatalf("flop didn't close: %q", s.Street)
	}
	street(t, s)
	if s.Street != Turn {
		t.Fatalf("street %q", s.Street)
	}
	// the next hand is a normal one, with the button moved on
	for s.InHand() {
		if s.Betting {
			act(t, s, s.ToAct, Check{})
		} else {
			street(t, s)
		}
	}
	start(t, s)
	if s.Bomb != 0 || s.Button != 2 || s.Street != Preflop {
		t.Fatalf("next hand bomb %d button %d street %q", s.Bomb, s.Button, s.Street)
	}
}

func TestBombShortStack(t *testing.T) {
	s := game(t, 30, 1000, 1000)
	bombHand(t, s, 50)
	if p := s.Player(1); p.Stack != 0 || p.Committed != 30 {
		t.Fatalf("short stack %+v", p)
	}
	if len(s.Pots) != 2 || s.Pots[0].Amount != 90 || s.Pots[1].Amount != 40 || len(s.Pots[1].Eligible) != 2 {
		t.Fatalf("pots %+v", s.Pots)
	}
	if !s.Betting || s.ToAct != 2 {
		t.Fatalf("seats 2 and 3 should bet, seat %d to act", s.ToAct)
	}
}

func TestBombAllIn(t *testing.T) {
	s := game(t, 30, 40)
	events := bombHand(t, s, 50)
	// seat 2's extra 10 goes back, and both hands turn up
	var shown int
	for _, e := range events {
		switch e := e.(type) {
		case Returned:
			if e.Seat != 2 || e.Amount != 10 {
				t.Fatalf("returned %+v", e)
			}
		case Shown:
			shown++
		}
	}
	if shown != 2 || s.Betting || s.Street != Flop {
		t.Fatalf("shown %d betting %v street %q", shown, s.Betting, s.Street)
	}
	street(t, s)
	street(t, s)
	if s.InHand() || s.Player(1).Stack+s.Player(2).Stack != 70 {
		t.Fatalf("hand over %v, stacks %d %d", !s.InHand(), s.Player(1).Stack, s.Player(2).Stack)
	}
}

func TestBombHeadsUp(t *testing.T) {
	s := game(t, 1000, 1000)
	bombHand(t, s, 20)
	// the button acts last after the flop
	if s.ToAct == s.Button {
		t.Fatalf("the button, seat %d, acts first", s.ToAct)
	}
}

func TestBombNoStraddles(t *testing.T) {
	s := game(t, 1000, 1000, 1000, 1000)
	if _, err := s.StartHand(Deal{Straddles: 1, Bomb: 20}); !errors.Is(err, ErrIllegal) {
		t.Fatalf("straddled a bomb pot: %v", err)
	}
	s.Config.Ante = 5
	bombHand(t, s, 20)
	for _, p := range s.Players {
		if p.Committed != 20 {
			t.Fatalf("seat %d put in %d, want just the bomb", p.Seat, p.Committed)
		}
	}
}
