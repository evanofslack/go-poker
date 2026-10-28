package holdem

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
)

func cards(s string) []Card {
	var cs []Card
	for _, f := range strings.Fields(s) {
		cs = append(cs, eval.MustParseCardString(f))
	}
	return cs
}

// stackDeck makes every hand deal the given cards in order: hole cards two at
// a time by seat, then a five card board. Unlisted cards come after.
func stackDeck(t *testing.T, order string) {
	t.Helper()
	first := cards(order)
	n := len(first) - 5
	var chunks [][]Card
	for i := 0; i < n; i += 2 {
		chunks = append(chunks, first[i:i+2])
	}
	chunks = append(chunks, first[n:n+3], first[n+3:n+4], first[n+4:])
	old := shuffle
	shuffle = func([]Card, [32]byte, int) []Card {
		deck := slices.DeleteFunc(slices.Clone(eval.DefaultDeck), func(c Card) bool { return slices.Contains(first, c) })
		// draw takes cards off the end, so the first chunk goes last
		for i := len(chunks) - 1; i >= 0; i-- {
			deck = append(deck, chunks[i]...)
		}
		return deck
	}
	t.Cleanup(func() { shuffle = old })
}

// game seats stacks in seats 1, 2, 3... with 10/20 blinds.
func game(t *testing.T, stacks ...uint) *State {
	t.Helper()
	s := New(Config{SmallBlind: 10, BigBlind: 20}, [32]byte{1})
	for i, stack := range stacks {
		if err := s.Sit(i+1, stack); err != nil {
			t.Fatal(err)
		}
	}
	return &s
}

func must(t *testing.T, events []Event, err error) []Event {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func start(t *testing.T, s *State) []Event {
	t.Helper()
	events, err := s.StartHand(Deal{})
	return must(t, events, err)
}

func street(t *testing.T, s *State) []Event {
	t.Helper()
	events, err := s.NextStreet()
	return must(t, events, err)
}

func act(t *testing.T, s *State, seat int, a Action) []Event {
	t.Helper()
	events, err := s.Act(seat, a)
	return must(t, events, err)
}

func stacks(s *State) []uint {
	var out []uint
	for _, p := range s.Players {
		out = append(out, p.Stack)
	}
	return out
}

func TestBlindsAndOrder(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	if s.Button != 1 || s.SB != 2 || s.BB != 3 || s.ToAct != 1 {
		t.Fatalf("button %d sb %d bb %d to act %d, want 1 2 3 1", s.Button, s.SB, s.BB, s.ToAct)
	}
	if got := stacks(s); !slices.Equal(got, []uint{1000, 990, 980}) {
		t.Fatalf("stacks %v", got)
	}
	act(t, s, 1, Call{})
	act(t, s, 2, Call{})
	// the big blind gets the option
	if l := s.Legal(3); !l.Check || l.RaiseMin != 40 || l.RaiseMax != 1000 {
		t.Fatalf("big blind option %+v", l)
	}
	act(t, s, 3, Check{})
	if s.Betting || s.Street != Preflop {
		t.Fatal("preflop should be closed, waiting for the flop")
	}
	street(t, s)
	if s.Street != Flop || len(s.Board) != 3 || s.ToAct != 2 {
		t.Fatalf("flop: street %s board %d to act %d", s.Street, len(s.Board), s.ToAct)
	}
}

func TestSetConfigBetweenHands(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	if err := s.SetConfig(Config{SmallBlind: 25, BigBlind: 50}); !errors.Is(err, ErrInHand) {
		t.Fatalf("mid-hand: %v, want ErrInHand", err)
	}
	act(t, s, 1, Fold{})
	act(t, s, 2, Fold{})
	if err := s.SetConfig(Config{SmallBlind: 25, BigBlind: 50, Ante: 5}); err != nil {
		t.Fatal(err)
	}
	// 1000, 990, 1010 after the first hand; now seat 3 posts 25 and seat 1 posts 50, plus antes
	start(t, s)
	if got := stacks(s); !slices.Equal(got, []uint{945, 985, 980}) {
		t.Fatalf("stacks %v", got)
	}
	if l := s.Legal(s.ToAct); l.RaiseMin != 100 {
		t.Fatalf("min raise to %d, want 100", l.RaiseMin)
	}
}

func TestHeadsUpButtonPostsSmallBlind(t *testing.T) {
	s := game(t, 1000, 1000)
	start(t, s)
	if s.Button != 1 || s.SB != 1 || s.BB != 2 || s.ToAct != 1 {
		t.Fatalf("button %d sb %d bb %d to act %d, want 1 1 2 1", s.Button, s.SB, s.BB, s.ToAct)
	}
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	street(t, s)
	if s.ToAct != 2 {
		t.Fatalf("big blind should act first after the flop, got %d", s.ToAct)
	}
}

func TestButtonMovesAndSkipsSittingOut(t *testing.T) {
	s := game(t, 1000, 1000, 1000, 1000)
	start(t, s)
	for s.InHand() {
		act(t, s, s.ToAct, Fold{})
	}
	if err := s.SetSittingOut(2, true); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	if s.Button != 3 || s.SB != 4 || s.BB != 1 {
		t.Fatalf("button %d sb %d bb %d, want 3 4 1", s.Button, s.SB, s.BB)
	}
	if s.Player(2).In {
		t.Fatal("sitting out player was dealt in")
	}
}

func TestMinRaise(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	if _, err := s.Act(1, Raise{To: 30}); !errors.Is(err, ErrIllegal) {
		t.Fatalf("raise to 30 should be under the min of 40, got %v", err)
	}
	act(t, s, 1, Raise{To: 60}) // a raise of 40
	if l := s.Legal(2); l.RaiseMin != 100 {
		t.Fatalf("next min raise to %d, want 100", l.RaiseMin)
	}
	if _, err := s.Act(2, Raise{To: 90}); !errors.Is(err, ErrIllegal) {
		t.Fatal("raise smaller than the last raise allowed")
	}
}

func TestShortAllInDoesNotReopen(t *testing.T) {
	s := game(t, 1000, 1000, 130)
	start(t, s)
	act(t, s, 1, Raise{To: 100})
	act(t, s, 2, Call{})
	// 130 all in is a raise of 30, less than the 80 minimum
	act(t, s, 3, Raise{To: 130})
	l := s.Legal(1)
	if l.Call != 30 || l.RaiseMax != 0 {
		t.Fatalf("player 1 after a short all in: %+v, want call 30 and no raise", l)
	}
}

func TestFoldWin(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	act(t, s, 1, Fold{})
	events := act(t, s, 2, Fold{})
	if s.InHand() {
		t.Fatal("hand should be over")
	}
	if got := stacks(s); !slices.Equal(got, []uint{1000, 990, 1010}) {
		t.Fatalf("stacks %v", got)
	}
	won, ok := events[1].(PotWon)
	if !ok || won.Amount != 30 || !slices.Equal(won.Winners, []int{3}) || won.Hand != "" {
		t.Fatalf("events %+v", events)
	}
	for _, p := range s.Players {
		if p.Shown {
			t.Fatal("no cards are shown in a fold win")
		}
	}
}

func TestShowdownSidePots(t *testing.T) {
	// seat 1 has aces, seat 2 kings, seat 3 queens; seat 1 is shortest
	stackDeck(t, "As Ad Ks Kd Qs Qd 2c 7h 9d 3s 4h")
	s := game(t, 100, 300, 500)
	start(t, s)
	act(t, s, 1, Raise{To: 100}) // all in
	act(t, s, 2, Raise{To: 300}) // all in
	act(t, s, 3, Call{})
	if s.Betting || !s.Player(1).Shown {
		t.Fatal("everyone is all in: betting closes with cards face up")
	}
	for s.InHand() {
		street(t, s)
	}
	// main 300 to aces, side 400 to kings over queens
	if got := stacks(s); !slices.Equal(got, []uint{300, 400, 200}) {
		t.Fatalf("stacks %v", got)
	}
	if s.Pots[0].Runs[0].Hand != "a pair of aces" || s.Pots[1].Runs[0].Hand != "a pair of kings" {
		t.Fatalf("pots %+v", s.Pots)
	}
}

func TestBustedSitsOut(t *testing.T) {
	stackDeck(t, "2c 7d As Ad 3h 8s 9d Jc Qh")
	s := game(t, 100, 1000)
	start(t, s)
	act(t, s, 1, Raise{To: 100})
	act(t, s, 2, Call{})
	for s.InHand() {
		street(t, s)
	}
	if !s.Player(1).SittingOut || s.Player(2).SittingOut {
		t.Fatal("the busted player should sit out")
	}
	if _, err := s.StartHand(Deal{}); !errors.Is(err, ErrNotEnoughPlayers) {
		t.Fatalf("one player left with chips: %v", err)
	}
}

func TestSplitPotOddChip(t *testing.T) {
	// both play the board's broadway straight
	stackDeck(t, "2c 3d 2d 3c 4s 4h Ah Kd Qc Js Th")
	s := game(t, 1000, 1000, 1000)
	start(t, s)
	act(t, s, 1, Fold{})
	act(t, s, 2, Raise{To: 40})
	act(t, s, 3, Call{})
	for s.InHand() {
		street(t, s)
		if s.Betting {
			act(t, s, s.ToAct, Check{})
			act(t, s, s.ToAct, Check{})
		}
	}
	// 80 split evenly
	if got := stacks(s); !slices.Equal(got, []uint{1000, 1000, 1000}) {
		t.Fatalf("stacks %v", got)
	}
}

func TestOddChipLeftOfButton(t *testing.T) {
	// seats 1 and 3 both play the board; seat 2 folds its small blind
	stackDeck(t, "2c 3d 7h 8h 2d 3c Ah Kd Qc Js Th")
	g := New(Config{SmallBlind: 5, BigBlind: 10}, [32]byte{})
	s := &g
	for seat := 1; seat <= 3; seat++ {
		s.Sit(seat, 1000)
	}
	start(t, s)
	act(t, s, 1, Raise{To: 20})
	act(t, s, 2, Fold{})
	act(t, s, 3, Call{})
	for s.InHand() {
		street(t, s)
		for s.Betting {
			act(t, s, s.ToAct, Check{})
		}
	}
	// a 45 chip pot splits 22 each, the odd chip to seat 3, first left of the button
	if got := stacks(s); !slices.Equal(got, []uint{1002, 995, 1003}) {
		t.Fatalf("stacks %v", got)
	}
}

func TestDeadMoneyInSidePot(t *testing.T) {
	// seat 1 short all in with the best hand; seats 2 and 3 build a side pot
	// and seat 3 folds to a bet on the flop
	stackDeck(t, "As Ad Ks Kd Qs Qd 2c 7h 9d 3s 4h")
	s := game(t, 50, 1000, 1000)
	start(t, s)
	act(t, s, 1, Raise{To: 50}) // all in
	act(t, s, 2, Raise{To: 200})
	act(t, s, 3, Call{})
	street(t, s)
	act(t, s, 2, Raise{To: 100})
	act(t, s, 3, Fold{})
	for s.InHand() {
		street(t, s)
	}
	// main 150 to aces; side 300 (150 dead from seat 3) to kings; the flop bet comes back
	if got := stacks(s); !slices.Equal(got, []uint{150, 1100, 800}) {
		t.Fatalf("stacks %v, pots %+v", got, s.Pots)
	}
}

func TestBlindAllIn(t *testing.T) {
	s := game(t, 1000, 5, 1000)
	start(t, s)
	if p := s.Player(2); !p.allIn() || p.Bet != 5 {
		t.Fatalf("small blind should be all in for 5: %+v", p)
	}
	act(t, s, 1, Call{})
	act(t, s, 3, Check{})
	if s.Betting {
		t.Fatal("preflop should close")
	}
	street(t, s)
	if s.ToAct != 3 {
		t.Fatalf("the all in small blind can't act, got %d", s.ToAct)
	}
}

func TestNoBettingAgainstNobody(t *testing.T) {
	// seat 2 goes all in, seat 1 calls with chips left: nobody to bet against
	s := game(t, 1000, 500, 1000)
	start(t, s)
	act(t, s, 1, Fold{})
	act(t, s, 2, Raise{To: 500})
	act(t, s, 3, Call{})
	if s.Betting {
		t.Fatal("betting should close")
	}
	for range 3 {
		street(t, s)
		if s.Betting {
			t.Fatal("no betting with one player left who has chips")
		}
	}
	if s.InHand() {
		t.Fatal("the river should go straight to showdown")
	}
}

func TestOutOfTurnAndBetweenHands(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	if _, err := s.Act(1, Check{}); !errors.Is(err, ErrNotYourTurn) {
		t.Fatalf("acting between hands: %v", err)
	}
	start(t, s)
	if _, err := s.Act(2, Fold{}); !errors.Is(err, ErrNotYourTurn) {
		t.Fatalf("acting out of turn: %v", err)
	}
	if err := s.Leave(1); !errors.Is(err, ErrInHand) {
		t.Fatalf("leaving mid-hand: %v", err)
	}
	if err := s.AddChips(1, 100); !errors.Is(err, ErrInHand) {
		t.Fatalf("adding chips mid-hand: %v", err)
	}
	if _, err := s.StartHand(Deal{}); !errors.Is(err, ErrInHand) {
		t.Fatalf("starting a hand mid-hand: %v", err)
	}
	act(t, s, 1, Fold{})
	// folded, but their chips in the pot don't leave with them
	if err := s.AddChips(1, 100); err != nil {
		t.Fatalf("a folded player who put nothing in can add chips: %v", err)
	}
	if err := s.AddChips(2, 100); !errors.Is(err, ErrInHand) {
		t.Fatalf("the small blind is still in: %v", err)
	}
}

func TestNotEnoughPlayers(t *testing.T) {
	s := game(t, 1000, 0)
	if _, err := s.StartHand(Deal{}); !errors.Is(err, ErrNotEnoughPlayers) {
		t.Fatalf("got %v", err)
	}
	if err := s.SetSittingOut(2, false); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("sitting in without chips: %v", err)
	}
}

func TestViewHidesCards(t *testing.T) {
	s := game(t, 1000, 1000)
	events := start(t, s)
	v := s.View(1)
	if hidden(v.Players[0].Cards) || !hidden(v.Players[1].Cards) {
		t.Fatal("player 1 should see only their own cards")
	}
	if spec := s.View(0); !hidden(spec.Players[0].Cards) {
		t.Fatal("spectators see no hole cards")
	}
	for _, e := range events {
		if d, ok := e.(Dealt); ok && Visible(e, 3-d.Seat) {
			t.Fatal("hole cards visible to the other player")
		}
	}
}

func TestSitAndLeave(t *testing.T) {
	s := game(t)
	for _, seat := range []int{5, 2, 9} {
		if err := s.Sit(seat, 100); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Sit(2, 100); !errors.Is(err, ErrSeatTaken) {
		t.Fatal("sat in a taken seat")
	}
	var seats []int
	for _, p := range s.Players {
		seats = append(seats, p.Seat)
	}
	if !slices.Equal(seats, []int{2, 5, 9}) {
		t.Fatalf("players not in seat order: %v", seats)
	}
	if err := s.Leave(5); err != nil || s.Player(5) != nil {
		t.Fatal("leave failed")
	}
}

// hidden reports whether cards are face down, or none.
func hidden(cards []Card) bool {
	return !slices.ContainsFunc(cards, func(c Card) bool { return c != 0 })
}

// TestNoSmallBlind plays antes and a big blind: nobody posts a small blind,
// and the seat after the button still acts after the big blind's left.
func TestNoSmallBlind(t *testing.T) {
	s := game(t, 1000, 1000, 1000)
	s.Config.SmallBlind, s.Config.Ante = 0, 5
	events := start(t, s)
	for _, e := range events {
		if a, ok := e.(Acted); ok && a.Kind == KindSmallBlind {
			t.Fatalf("posted %+v", a)
		}
	}
	// button 1, the small blind's seat 2 posts only the ante, big blind 3
	if p := s.Player(2); p.Committed != 5 || p.Bet != 0 {
		t.Fatalf("seat 2 put in %d, bet %d", p.Committed, p.Bet)
	}
	if s.ToAct != 1 || s.Legal(1).Call != 20 {
		t.Fatalf("seat %d to act, legal %+v", s.ToAct, s.Legal(1))
	}
	// everyone limps, and the big blind checks to close it
	act(t, s, 1, Call{})
	act(t, s, 2, Call{})
	act(t, s, 3, Check{})
	if s.Betting || s.Pots[0].Amount != 75 {
		t.Fatalf("betting %v, pots %+v", s.Betting, s.Pots)
	}
}

func TestNoSmallBlindHeadsUp(t *testing.T) {
	s := game(t, 1000, 1000)
	s.Config.SmallBlind = 0
	start(t, s)
	// the button posts nothing and acts first
	if s.ToAct != s.Button || s.Legal(s.Button).Call != 20 || s.Player(s.Button).Committed != 0 {
		t.Fatalf("seat %d to act, button %d, legal %+v", s.ToAct, s.Button, s.Legal(s.ToAct))
	}
}
