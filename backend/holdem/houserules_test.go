package holdem

import (
	"errors"
	"slices"
	"testing"
)

// anteGame seats stacks in seats 1, 2, 3... with 10/20 blinds and an ante.
func anteGame(t *testing.T, ante uint, stacks ...uint) *State {
	t.Helper()
	s := New(Config{SmallBlind: 10, BigBlind: 20, Ante: ante}, [32]byte{1})
	for i, stack := range stacks {
		if err := s.Sit(i+1, stack); err != nil {
			t.Fatal(err)
		}
	}
	return &s
}

// ok fails t if the call it wraps fails, and returns its events.
func ok(t *testing.T) func([]Event, error) []Event {
	return func(events []Event, err error) []Event { return must(t, events, err) }
}

func potTotal(s *State) uint {
	var n uint
	for _, p := range s.Pots {
		n += p.Amount
	}
	return n
}

func TestAntes(t *testing.T) {
	s := anteGame(t, 5, 1000, 1000, 1000)
	events := start(t, s)
	if got := stacks(s); !slices.Equal(got, []uint{995, 985, 975}) {
		t.Fatalf("stacks %v", got)
	}
	if potTotal(s) != 45 {
		t.Fatalf("pot %d, want 45", potTotal(s))
	}
	// antes are dead money: the call and min raise are as without them
	if l := s.Legal(1); l.Call != 20 || l.RaiseMin != 40 {
		t.Fatalf("legal %+v", l)
	}
	if p := s.Player(1); p.Bet != 0 || p.Committed != 5 {
		t.Fatalf("button bet %d committed %d", p.Bet, p.Committed)
	}
	antes := 0
	for _, e := range events {
		if a, ok := e.(Acted); ok && a.Kind == KindAnte && a.Amount == 5 {
			antes++
		}
	}
	if antes != 3 {
		t.Fatalf("%d antes in %+v", antes, events)
	}
}

func TestShortAnteAllIn(t *testing.T) {
	s := anteGame(t, 5, 3, 1000, 1000)
	start(t, s)
	if p := s.Player(1); !p.allIn() || p.Committed != 3 {
		t.Fatalf("seat 1 should be all in on the ante: %+v", p)
	}
	// a main pot of three antes, the rest in a side pot seat 1 can't win
	if len(s.Pots) != 2 || s.Pots[0].Amount != 9 || !slices.Equal(s.Pots[0].Eligible, []int{1, 2, 3}) ||
		s.Pots[1].Amount != 34 || slices.Contains(s.Pots[1].Eligible, 1) {
		t.Fatalf("pots %+v", s.Pots)
	}
	if s.ToAct != 2 {
		t.Fatalf("to act %d, want the small blind", s.ToAct)
	}
}

func TestEveryoneAllInOnAntes(t *testing.T) {
	s := anteGame(t, 20, 5, 5)
	start(t, s)
	if s.Betting {
		t.Fatal("nobody has chips to bet")
	}
	for s.InHand() {
		street(t, s)
	}
	if got := stacks(s); got[0]+got[1] != 10 {
		t.Fatalf("stacks %v", got)
	}
}

// checkDown plays a heads-up hand to showdown, checking every street. Seat 1
// has aces, seat 2 kings, and river is the river action, if any.
func checkDown(t *testing.T, river func(s *State)) *State {
	t.Helper()
	stackDeck(t, "As Ad Ks Kd 2c 7h 9d 3s 4h")
	s := game(t, 1000, 1000)
	start(t, s)
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	for range 2 {
		street(t, s)
		act(t, s, 2, Check{})
		act(t, s, 1, Check{})
	}
	street(t, s)
	river(s)
	if s.InHand() {
		t.Fatal("the hand should be over")
	}
	return s
}

func TestCheckedDownLoserMucks(t *testing.T) {
	s := checkDown(t, func(s *State) {
		act(t, s, 2, Check{})
		act(t, s, 1, Check{})
	})
	if !s.Player(1).Shown || s.Player(2).Shown {
		t.Fatal("only the winner shows")
	}
	if v := s.View(1); v.Players[1].Cards != [2]Card{} {
		t.Fatal("a mucked hand reached another player")
	}
	if v := s.View(2); v.Players[1].Cards == [2]Card{} {
		t.Fatal("you still see your own mucked hand")
	}
}

func TestCalledLoserShows(t *testing.T) {
	s := checkDown(t, func(s *State) {
		act(t, s, 2, Raise{To: 40})
		act(t, s, 1, Call{})
	})
	if !s.Player(1).Shown || !s.Player(2).Shown {
		t.Fatal("the called bettor and the winner show")
	}
}

func TestCallerMucks(t *testing.T) {
	s := checkDown(t, func(s *State) {
		act(t, s, 2, Check{})
		act(t, s, 1, Raise{To: 40})
		act(t, s, 2, Call{})
	})
	if !s.Player(1).Shown || s.Player(2).Shown {
		t.Fatal("the winning bettor shows, the losing caller mucks")
	}
}

func TestShow(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	if _, err := s.Show(1); !errors.Is(err, ErrInHand) {
		t.Fatalf("show mid-hand: %v", err)
	}
	act(t, s, 1, Fold{})
	act(t, s, 2, Fold{})

	events := ok(t)(s.Show(3))
	if sh, ok := events[0].(Shown); !ok || sh.Seat != 3 || s.View(1).Players[2].Cards == [2]Card{} {
		t.Fatalf("the winner's cards should be face up: %+v", events)
	}
	if _, err := s.Show(3); !errors.Is(err, ErrIllegal) {
		t.Fatalf("showing twice: %v", err)
	}
	// folded hands can be shown too
	ok(t)(s.Show(1))

	if err := s.Sit(4, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Show(4); !errors.Is(err, ErrIllegal) {
		t.Fatalf("show without being dealt in: %v", err)
	}
	start(t, s)
	if s.Player(3).Shown {
		t.Fatal("a new hand starts face down")
	}
}

func TestHunt(t *testing.T) {
	stackDeck(t, "As Ad Ks Kd 2c 7h 9d 3s 4h")
	s := game(t, 1000, 1000)
	start(t, s)
	if _, err := s.Hunt(); !errors.Is(err, ErrIllegal) {
		t.Fatalf("hunt mid-hand: %v", err)
	}
	act(t, s, 1, Fold{})

	events := ok(t)(s.Hunt())
	want := cards("2c 7h 9d 3s 4h")
	if !slices.Equal(s.Hunted, want) || !slices.Equal(events[0].(Hunted).Cards, want) {
		t.Fatalf("hunted %v, want the board that would have come", s.Hunted)
	}
	if !slices.Equal(s.View(0).Hunted, want) {
		t.Fatal("everyone sees the hunt")
	}
	if len(s.Board) != 0 || len(s.Pots[0].Winners) != 1 {
		t.Fatal("the hunt counts for nothing")
	}
	if _, err := s.Hunt(); !errors.Is(err, ErrIllegal) {
		t.Fatalf("hunting twice: %v", err)
	}

	start(t, s)
	if s.Hunted != nil {
		t.Fatal("a new hand clears the hunt")
	}
}

func TestHuntAfterFlop(t *testing.T) {
	stackDeck(t, "As Ad Ks Kd 2c 7h 9d 3s 4h")
	s := game(t, 1000, 1000)
	start(t, s)
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	street(t, s)
	act(t, s, 2, Raise{To: 20})
	act(t, s, 1, Fold{})
	ok(t)(s.Hunt())
	if !slices.Equal(s.Hunted, cards("3s 4h")) {
		t.Fatalf("hunted %v, want the turn and river", s.Hunted)
	}
}

func TestNoHuntAfterShowdown(t *testing.T) {
	s := checkDown(t, func(s *State) {
		act(t, s, 2, Check{})
		act(t, s, 1, Check{})
	})
	if _, err := s.Hunt(); !errors.Is(err, ErrIllegal) {
		t.Fatalf("hunt with a full board: %v", err)
	}
}
