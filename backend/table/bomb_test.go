package table

import (
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// bombTable seats a, b and c with 1000 each, a hosting, with bomb pots of 50.
func bombTable(t *testing.T, edit func(*Settings)) *harness {
	set := DefaultSettings
	set.BombPot = 50
	if edit != nil {
		edit(&set)
	}
	h := newHarness(t, set, "")
	for i, name := range []string{"a", "b", "c"} {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	return h
}

func TestBombPotNextHand(t *testing.T) {
	h := bombTable(t, nil)
	if msg := h.refused(BombPot{Person: h.ids["b"], On: true}); msg != errNotHost.Error() {
		t.Fatalf("b called a bomb pot: %q", msg)
	}
	h.ok(BombPot{Person: h.ids["a"], On: true})
	h.mustLog("a makes the next hand a bomb pot of 50 each")
	if !ViewFor(h.s, h.ids["b"], h.now).BombNext {
		t.Fatal("b doesn't see the bomb pot coming")
	}
	h.refused(BombPot{Person: h.ids["a"], On: true})
	h.ok(Start{Person: h.ids["a"]})
	g := h.s.Game
	if g.Bomb != 50 || g.Street != holdem.Flop || len(g.Board) != 3 || h.s.BombNext {
		t.Fatalf("bomb %d street %q board %v next %v", g.Bomb, g.Street, g.Board, h.s.BombNext)
	}
	if v := ViewFor(h.s, h.ids["b"], h.now); v.Bomb != 50 {
		t.Fatalf("view bomb %d", v.Bomb)
	}
	for h.s.Game.InHand() {
		if h.s.Game.Betting {
			h.play(holdem.Check{})
		} else {
			h.advance(time.Second)
		}
	}
	rec := h.s.History[0]
	if rec.Bomb != 50 || rec.Actions[0].Kind != "bomb" || len(rec.Actions) != 3+9 {
		t.Fatalf("record bomb %d actions %+v", rec.Bomb, rec.Actions)
	}
	if tot := h.s.Totals["a"]; tot.Hands != 1 || tot.Bombs != 1 || tot.VPIP != 0 || tot.SawFlop != 1 {
		t.Fatalf("totals %+v", tot)
	}
	// the next one is a normal hand
	h.nextHand()
	if h.s.Game.Bomb != 0 || h.s.Game.Street != holdem.Preflop {
		t.Fatalf("hand 2 bomb %d street %q", h.s.Game.Bomb, h.s.Game.Street)
	}
}

func TestBombPotCancel(t *testing.T) {
	h := bombTable(t, nil)
	h.refused(BombPot{Person: h.ids["a"]})
	h.ok(BombPot{Person: h.ids["a"], On: true})
	h.ok(BombPot{Person: h.ids["a"]})
	h.mustLog("a called off the bomb pot")
	h.ok(Start{Person: h.ids["a"]})
	if h.s.Game.Bomb != 0 {
		t.Fatal("dealt a bomb pot after calling it off")
	}

	set := DefaultSettings
	h = newHarness(t, set, "")
	h.join("a")
	if msg := h.refused(BombPot{Person: h.ids["a"], On: true}); msg != "set a bomb pot amount first" {
		t.Fatalf("no amount: %q", msg)
	}
}

func TestBombPotEvery(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.BombEvery = 3 })
	h.ok(Start{Person: h.ids["a"]})
	var bombs []int
	for hand := 1; hand <= 6; hand++ {
		if h.s.Game.Bomb > 0 {
			bombs = append(bombs, hand)
		}
		for h.s.Game.InHand() {
			if h.s.Game.Betting {
				h.play(holdem.Fold{})
			} else {
				h.advance(time.Second)
			}
		}
		if hand < 6 {
			h.nextHand()
		}
	}
	if !slices.Equal(bombs, []int{3, 6}) {
		t.Fatalf("bomb pots on hands %v", bombs)
	}
}

func TestBombPotSkipsStraddles(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.Straddle = StraddleForced })
	h.ok(BombPot{Person: h.ids["a"], On: true})
	if h.s.StraddleAsk != nil {
		t.Fatalf("asked to straddle a bomb pot: %+v", h.s.StraddleAsk)
	}
	h.ok(Start{Person: h.ids["a"]})
	if h.s.Game.Bomb != 50 || len(h.s.Game.Straddles) != 0 {
		t.Fatalf("bomb %d straddles %v", h.s.Game.Bomb, h.s.Game.Straddles)
	}
}

func TestDoubleBoardTable(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.DoubleBoard = true })
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.Double || h.s.Game.Bomb != 0 {
		t.Fatalf("double %v bomb %d", h.s.Game.Double, h.s.Game.Bomb)
	}
	for !h.s.Game.Betting || h.s.Game.Street == holdem.Preflop {
		if h.s.Game.Betting {
			if l := h.s.Game.Legal(h.s.Game.ToAct); l.Check {
				h.play(holdem.Check{})
			} else {
				h.play(holdem.Call{})
			}
		} else {
			h.advance(time.Second)
		}
	}
	if v := ViewFor(h.s, h.ids["a"], h.now); len(v.MadeHand) != 2 || len(v.Boards) != 2 {
		t.Fatalf("a sees %v on %v", v.MadeHand, v.Boards)
	}
}

func TestDoubleBoardBombTable(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.BombDouble = true })
	h.ok(BombPot{Person: h.ids["a"], On: true})
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.Double || h.s.Game.Bomb != 50 {
		t.Fatalf("double %v bomb %d", h.s.Game.Double, h.s.Game.Bomb)
	}
	for h.s.Game.InHand() {
		if h.s.Game.Betting {
			h.play(holdem.Check{})
		} else {
			h.advance(time.Second)
		}
	}
	if rec := h.s.History[0]; len(rec.Boards) != 2 || len(rec.Boards[1]) != 5 {
		t.Fatalf("recorded boards %v", rec.Boards)
	}
}

func TestSevenDeuceSecondBoard(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.DoubleBoard, s.SevenDeuce = true, 40 })
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(1).Cards = hole("Ah Ad")
	h.s.Game.Player(2).Cards = hole("7c 2d")
	h.s.Game.Player(3).Cards = hole("3c 4d")
	for h.s.Game.Street != holdem.River || !h.s.Game.Betting {
		if h.s.Game.Betting {
			if h.s.Game.Legal(h.s.Game.ToAct).Check {
				h.play(holdem.Check{})
			} else {
				h.play(holdem.Call{})
			}
		} else {
			h.advance(time.Second)
		}
	}
	// a wins board one, b fills up on board two
	h.s.Game.Boards = [][]holdem.Card{board("Kc Qd 9h 5s 3h"), board("7h 7s 2s Kd Js")}
	h.s.Game.Board = h.s.Game.Boards[0]
	for h.s.Game.InHand() {
		h.play(holdem.Check{})
	}
	if !slices.ContainsFunc(h.s.History[0].Payments, func(p Payment) bool { return p.Kind == KindSevenDeuce && p.To == "b" }) {
		t.Fatalf("payments %+v", h.s.History[0].Payments)
	}
}

func TestBombSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*Settings)
	}{
		{"bomb too big", func(s *Settings) { s.BombPot = 21 * s.BigBlind }},
		{"every with no bomb", func(s *Settings) { s.BombEvery = 5 }},
		{"every too far apart", func(s *Settings) { s.BombPot, s.BombEvery = 20, 101 }},
		{"two PLO5 boards for 9", func(s *Settings) { s.Seats, s.Game, s.DoubleBoard = 9, GameOmaha5, true }},
		{"two PLO5 bomb boards for 9", func(s *Settings) { s.Seats, s.Game, s.BombPot, s.BombDouble = 9, GameOmaha5, 20, true }},
	} {
		set := DefaultSettings
		tt.edit(&set)
		if set.Validate() == nil {
			t.Errorf("%s: valid", tt.name)
		}
	}
	ok := DefaultSettings
	ok.Seats, ok.Game, ok.DoubleBoard = 8, GameOmaha5, true
	if err := ok.Validate(); err != nil {
		t.Fatalf("PLO5 double board for 8: %v", err)
	}

	h := bombTable(t, nil)
	h.ok(BombPot{Person: h.ids["a"], On: true})
	set := DefaultSettings
	set.BombEvery = 0
	set.DoubleBoard = true
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("the bomb pot to off, double board on")
	if h.s.BombNext {
		t.Fatal("bomb pot still next with none set")
	}
}
