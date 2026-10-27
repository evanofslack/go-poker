package holdem

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
)

// stackDraws makes every hand deal draws in order, one entry per draw: hole
// cards by seat, then the flop, turn, river and any further runs.
func stackDraws(t *testing.T, draws ...string) {
	t.Helper()
	var listed []Card
	for _, d := range draws {
		listed = append(listed, cards(d)...)
	}
	old := shuffle
	shuffle = func([32]byte, int) []Card {
		deck := slices.DeleteFunc(slices.Clone(eval.DefaultDeck), func(c Card) bool { return slices.Contains(listed, c) })
		// draw takes cards off the end, so the first draw goes last
		for i := len(draws) - 1; i >= 0; i-- {
			deck = append(deck, cards(draws[i])...)
		}
		return deck
	}
	t.Cleanup(func() { shuffle = old })
}

// allInPreflop has seat 1, on the button, shove and seat 2 call, heads up.
func allInPreflop(t *testing.T, s *State) {
	t.Helper()
	start(t, s)
	act(t, s, 1, Raise{To: s.Legal(1).RaiseMax})
	act(t, s, 2, Call{})
	if s.Betting || !s.InHand() {
		t.Fatal("both should be all in, waiting for the flop")
	}
}

// runOut deals the rest of the hand, returning what it did.
func runOut(t *testing.T, s *State) []Event {
	t.Helper()
	var events []Event
	for s.InHand() {
		events = append(events, street(t, s)...)
	}
	return events
}

func TestMaxRuns(t *testing.T) {
	s := game(t, 1000, 1000)
	start(t, s)
	if n := s.MaxRuns(); n != 0 {
		t.Fatalf("while betting: %d, want 0", n)
	}
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	if n := s.MaxRuns(); n != 0 {
		t.Fatalf("two can still bet: %d, want 0", n)
	}

	s = game(t, 1000, 1000)
	allInPreflop(t, s)
	if n := s.MaxRuns(); n != 9 {
		t.Fatalf("heads up preflop: %d, want 9", n) // 48 cards, 5 a run
	}

	// the big stack has chips left, but nobody to bet against
	s = game(t, 2000, 1000)
	start(t, s)
	act(t, s, 1, Raise{To: 2000})
	act(t, s, 2, Call{})
	if n := s.MaxRuns(); n != 9 {
		t.Fatalf("covered preflop: %d, want 9", n)
	}

	s = game(t, 1000, 1000)
	start(t, s)
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	street(t, s)
	act(t, s, 2, Raise{To: 980})
	act(t, s, 1, Call{})
	if n := s.MaxRuns(); n != 22 {
		t.Fatalf("on the flop: %d, want 22", n) // 45 cards, 2 a run
	}
	street(t, s)
	if n := s.MaxRuns(); n != 44 {
		t.Fatalf("on the turn: %d, want 44", n)
	}
}

func TestSetRunsIllegal(t *testing.T) {
	s := game(t, 1000, 1000)
	start(t, s)
	before := s.Clone()
	if err := s.SetRuns(2); !errors.Is(err, ErrIllegal) || !reflect.DeepEqual(*s, before) {
		t.Fatalf("while betting: %v", err)
	}

	s = game(t, 1000, 1000)
	allInPreflop(t, s)
	for _, n := range []int{0, 10} {
		before := s.Clone()
		if err := s.SetRuns(n); !errors.Is(err, ErrIllegal) || !reflect.DeepEqual(*s, before) {
			t.Fatalf("SetRuns(%d): %v", n, err)
		}
	}
	if err := s.SetRuns(2); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRuns(3); !errors.Is(err, ErrIllegal) {
		t.Fatalf("a second time: %v", err)
	}
}

func TestRunTwiceSameWinner(t *testing.T) {
	stackDraws(t, "Ah Ad", "7c 2d", "3s 4s 9h", "Jd", "Kc", "5c 6d 8h Th Qs")
	s := game(t, 1000, 1000)
	allInPreflop(t, s)
	if err := s.SetRuns(2); err != nil {
		t.Fatal(err)
	}
	events := runOut(t, s)
	if got := stacks(s); !slices.Equal(got, []uint{2000, 0}) {
		t.Fatalf("stacks %v", got)
	}
	pot := s.Pots[0]
	if len(pot.Runs) != 2 || pot.Runs[0].Amount != 1000 || pot.Runs[1].Amount != 1000 || !slices.Equal(pot.Winners, []int{1}) {
		t.Fatalf("pot %+v", pot)
	}

	var dealt, won []int
	for _, e := range events {
		switch e := e.(type) {
		case StreetDealt:
			dealt = append(dealt, e.Run)
		case PotWon:
			won = append(won, e.Run)
		}
	}
	if !slices.Equal(dealt, []int{0, 0, 0, 1}) || !slices.Equal(won, []int{0, 1}) {
		t.Fatalf("street runs %v, pot runs %v", dealt, won)
	}
}

func TestRunTwiceSplit(t *testing.T) {
	// aces hold on the first board; the second makes the 7-2 a full house
	stackDraws(t, "Ah Ad", "7c 2d", "3s 4s 9h", "Jd", "Kc", "7h 7s 2s Qc 8d")
	s := game(t, 1000, 1000)
	allInPreflop(t, s)
	if err := s.SetRuns(2); err != nil {
		t.Fatal(err)
	}
	runOut(t, s)
	if got := stacks(s); !slices.Equal(got, []uint{1000, 1000}) {
		t.Fatalf("stacks %v", got)
	}
	pot := s.Pots[0]
	if !slices.Equal(pot.Runs[0].Winners, []int{1}) || !slices.Equal(pot.Runs[1].Winners, []int{2}) || !slices.Equal(pot.Winners, []int{1, 2}) {
		t.Fatalf("pot %+v", pot)
	}
	if pot.Runs[0].Hand != "a pair of aces" || pot.Runs[1].Hand != "a full house, sevens full of twos" {
		t.Fatalf("hands %q, %q", pot.Runs[0].Hand, pot.Runs[1].Hand)
	}
	if len(s.Boards) != 2 || !slices.Equal(s.Boards[0], s.Board) || len(s.Boards[1]) != 5 {
		t.Fatalf("boards %v, board %v", s.Boards, s.Board)
	}
}

func TestRunsOddChipAndTie(t *testing.T) {
	// a 5 small blind folds, so the pot is odd: 2005 over two runs
	stackDraws(t, "Ah Ad", "9c 9d", "Kh Kd", "3s 4c 8h", "Jd", "2c", "Ts Js Qs Ks As")
	s := game(t, 1000, 1000, 1000)
	s.Config.SmallBlind = 5
	start(t, s)
	act(t, s, 1, Raise{To: 1000})
	act(t, s, 2, Fold{})
	act(t, s, 3, Call{})
	if err := s.SetRuns(2); err != nil {
		t.Fatal(err)
	}
	runOut(t, s)
	// aces take the first run's 1003; the board's royal flush splits the second's 1002
	if got := stacks(s); !slices.Equal(got, []uint{1003 + 501, 995, 501}) {
		t.Fatalf("stacks %v", got)
	}
	if runs := s.Pots[0].Runs; runs[0].Amount != 1003 || runs[1].Amount != 1002 || !slices.Equal(runs[1].Winners, []int{1, 3}) {
		t.Fatalf("runs %+v", runs)
	}
}

func TestRunsSidePot(t *testing.T) {
	// a is short, so the main pot is 1500 and the side pot 1000 between b and c
	stackDraws(t, "Ah Ad", "Kh Kd", "Qh Qd", "2s 5c 8d", "9c", "Jh", "Qs Qc 3d 4h 6s")
	s := game(t, 500, 1000, 1000)
	start(t, s)
	act(t, s, 1, Raise{To: 500})
	act(t, s, 2, Raise{To: 1000})
	act(t, s, 3, Call{})
	if err := s.SetRuns(2); err != nil {
		t.Fatal(err)
	}
	runOut(t, s)
	// first run: aces win the main, kings the side; second: quads win both
	if got := stacks(s); !slices.Equal(got, []uint{750, 500, 1250}) {
		t.Fatalf("stacks %v", got)
	}
	if len(s.Pots) != 2 || len(s.Pots[1].Runs) != 2 || !slices.Equal(s.Pots[1].Winners, []int{2, 3}) {
		t.Fatalf("pots %+v", s.Pots)
	}
}

func TestRunsFromTheFlop(t *testing.T) {
	s := game(t, 1000, 1000)
	start(t, s)
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	street(t, s)
	act(t, s, 2, Raise{To: 980})
	act(t, s, 1, Call{})
	if err := s.SetRuns(3); err != nil {
		t.Fatal(err)
	}
	runOut(t, s)
	if s.RunFrom != 3 || len(s.Boards) != 3 || len(s.Pots[0].Runs) != 3 {
		t.Fatalf("run from %d, boards %d, runs %d", s.RunFrom, len(s.Boards), len(s.Pots[0].Runs))
	}
	for _, b := range s.Boards {
		if !slices.Equal(b[:3], s.Board[:3]) {
			t.Fatalf("board %v doesn't share the flop %v", b, s.Board[:3])
		}
	}
	if s.Boards[1][3] == s.Boards[2][3] || s.Boards[0][4] == s.Boards[1][4] {
		t.Fatalf("runs reuse cards: %v", s.Boards)
	}
	if got := stacks(s); got[0]+got[1] != 2000 {
		t.Fatalf("stacks %v", got)
	}
}

func TestOneRunByDefault(t *testing.T) {
	s := game(t, 1000, 1000)
	allInPreflop(t, s)
	runOut(t, s)
	if len(s.Boards) != 1 || !slices.Equal(s.Boards[0], s.Board) || len(s.Pots[0].Runs) != 1 {
		t.Fatalf("boards %v, runs %+v", s.Boards, s.Pots[0].Runs)
	}
	if _, err := s.NextStreet(); !errors.Is(err, ErrIllegal) {
		t.Fatalf("after the hand: %v", err)
	}
}
