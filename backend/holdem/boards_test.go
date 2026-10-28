package holdem

import (
	"errors"
	"slices"
	"testing"
)

// playOut checks or calls until the hand ends.
func playOut(t *testing.T, s *State) []Event {
	t.Helper()
	var events []Event
	for s.InHand() {
		switch {
		case !s.Betting:
			events = append(events, street(t, s)...)
		case s.Legal(s.ToAct).Check:
			events = append(events, act(t, s, s.ToAct, Check{})...)
		default:
			events = append(events, act(t, s, s.ToAct, Call{})...)
		}
	}
	return events
}

func TestDoubleBoard(t *testing.T) {
	// seat 1 wins board one with a flush, seat 2 board two with a straight
	stackDraws(t, "Ah 2h", "9c Tc", "Kh 7h 3s", "Kd 8d Jd", "4h", "Qs", "2c", "3d")
	s := game(t, 1000, 1000)
	events, err := s.StartHand(Deal{Boards: 2})
	must(t, events, err)
	if !s.Double || s.Runs != 2 || s.MaxRuns() != 0 {
		t.Fatalf("double %v runs %d max runs %d", s.Double, s.Runs, s.MaxRuns())
	}
	act(t, s, s.ToAct, Call{})
	act(t, s, s.ToAct, Check{})
	flop := street(t, s)
	if len(flop) < 2 || flop[1].(StreetDealt).Run != 1 {
		t.Fatalf("flop events %+v", flop)
	}
	if !slices.Equal(s.Boards[0], cards("Kh 7h 3s")) || !slices.Equal(s.Boards[1], cards("Kd 8d Jd")) || !slices.Equal(s.Board, s.Boards[0]) {
		t.Fatalf("boards %v, board %v", s.Boards, s.Board)
	}
	playOut(t, s)
	if len(s.Boards) != 2 || len(s.Boards[0]) != 5 || len(s.Boards[1]) != 5 {
		t.Fatalf("boards %v", s.Boards)
	}
	pot := s.Pots[0]
	if len(pot.Runs) != 2 || pot.Runs[0].Winners[0] != 1 || pot.Runs[1].Winners[0] != 2 || pot.Runs[0].Amount != 20 {
		t.Fatalf("pot %+v", pot)
	}
	if s.Player(1).Stack != 1000 || s.Player(2).Stack != 1000 {
		t.Fatalf("stacks %v", stacks(s))
	}
	if _, err := s.Hunt(); !errors.Is(err, ErrIllegal) {
		t.Fatal("hunted a double board")
	}
}

func TestDoubleBoardScoop(t *testing.T) {
	// seat 1 has aces on both boards
	stackDraws(t, "As Ad", "7c 2d", "Kh 9h 3s", "Qd 8c 4d", "5h", "Jc", "6s", "Ts")
	s := game(t, 1000, 1000, 1000)
	{
		events, err := s.StartHand(Deal{Boards: 2})
		must(t, events, err)
	}
	playOut(t, s)
	if s.Player(1).Stack != 1040 {
		t.Fatalf("stacks %v, want seat 1 to scoop", stacks(s))
	}
}

func TestDoubleBoardAllIn(t *testing.T) {
	s := game(t, 100, 1000)
	{
		events, err := s.StartHand(Deal{Boards: 2})
		must(t, events, err)
	}
	act(t, s, s.ToAct, Raise{To: 100})
	act(t, s, s.ToAct, Call{})
	if s.MaxRuns() != 0 {
		t.Fatal("offered to run a double board more times")
	}
	playOut(t, s)
	if len(s.Boards) != 2 || len(s.Pots[0].Runs) != 2 || s.Player(1).Stack+s.Player(2).Stack != 1100 {
		t.Fatalf("boards %v pots %+v", s.Boards, s.Pots)
	}
	if seen := slices.Concat(s.Boards[0], s.Boards[1]); len(slices.Compact(slices.Sorted(slices.Values(seen)))) != 10 {
		t.Fatalf("boards share cards: %v", s.Boards)
	}
}

func TestDoubleBoardBomb(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	{
		events, err := s.StartHand(Deal{Bomb: 50, Boards: 2})
		must(t, events, err)
	}
	if s.Street != Flop || len(s.Boards) != 2 || len(s.Boards[1]) != 3 {
		t.Fatalf("street %q boards %v", s.Street, s.Boards)
	}
	playOut(t, s)
	var total uint
	for _, p := range s.Players {
		total += p.Stack
	}
	if total != 3000 {
		t.Fatalf("chips %d", total)
	}
}

func TestDealBoards(t *testing.T) {
	s := game(t, 1000, 1000)
	for _, n := range []int{-1, 3} {
		if _, err := s.StartHand(Deal{Boards: n}); !errors.Is(err, ErrIllegal) {
			t.Fatalf("%d boards: %v", n, err)
		}
	}
}

func TestDealTooBig(t *testing.T) {
	s := game(t, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000)
	s.Config.Game = Game{Hole: 5}
	if _, err := s.StartHand(Deal{Boards: 2}); !errors.Is(err, ErrIllegal) {
		t.Fatalf("dealt 55 cards: %v", err)
	}
	start(t, s) // one board fits: 50
}
