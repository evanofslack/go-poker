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
	if msg := h.refused(SpecialHand{Person: h.ids["b"], Kind: SpecialBomb}); msg != errNotHost.Error() {
		t.Fatalf("b called a bomb pot: %q", msg)
	}
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb})
	h.mustLog("a makes the next hand a bomb pot of 50 each")
	if ViewFor(h.s, h.ids["b"], h.now).SpecialNext != SpecialBomb {
		t.Fatal("b doesn't see the bomb pot coming")
	}
	h.refused(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb})
	h.ok(Start{Person: h.ids["a"]})
	g := h.s.Game
	if g.Bomb != 50 || g.Street != holdem.Flop || len(g.Board) != 3 || h.s.SpecialNext != "" {
		t.Fatalf("bomb %d street %q board %v next %q", g.Bomb, g.Street, g.Board, h.s.SpecialNext)
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
	h.refused(SpecialHand{Person: h.ids["a"]})
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb})
	h.ok(SpecialHand{Person: h.ids["a"]})
	h.mustLog("a called off the bomb pot")
	h.ok(Start{Person: h.ids["a"]})
	if h.s.Game.Bomb != 0 {
		t.Fatal("dealt a bomb pot after calling it off")
	}

	set := DefaultSettings
	h = newHarness(t, set, "")
	h.join("a")
	if msg := h.refused(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb}); msg != "bomb pots are off at this table" {
		t.Fatalf("no amount: %q", msg)
	}
}

func TestBombPotEvery(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.SpecialEvery = 3 })
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
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb})
	if h.s.StraddleAsk != nil {
		t.Fatalf("asked to straddle a bomb pot: %+v", h.s.StraddleAsk)
	}
	h.ok(Start{Person: h.ids["a"]})
	if h.s.Game.Bomb != 50 || len(h.s.Game.Straddles) != 0 {
		t.Fatalf("bomb %d straddles %v", h.s.Game.Bomb, h.s.Game.Straddles)
	}
}

func TestDoubleBoardBombTable(t *testing.T) {
	h := bombTable(t, nil)
	// double board comes only with a bomb pot
	if msg := h.refused(SpecialHand{Person: h.ids["a"], Kind: "double"}); msg != "that isn't a special hand" {
		t.Fatalf("double board alone: %q", msg)
	}
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialDoubleBomb})
	h.mustLog("a makes the next hand a double board bomb pot of 50 each")
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.Double || h.s.Game.Bomb != 50 {
		t.Fatalf("double %v bomb %d", h.s.Game.Double, h.s.Game.Bomb)
	}
	if v := ViewFor(h.s, h.ids["a"], h.now); len(v.MadeHand) != 2 || len(v.Boards) != 2 {
		t.Fatalf("a sees %v on %v", v.MadeHand, v.Boards)
	}
	for h.s.Game.InHand() {
		if h.s.Game.Betting {
			h.play(holdem.Check{})
		} else {
			h.advance(time.Second)
		}
	}
	if rec := h.s.History[0]; !rec.Double || len(rec.Boards) != 2 || len(rec.Boards[1]) != 5 {
		t.Fatalf("recorded boards %v", rec.Boards)
	}
}

func TestSevenDeuceSecondBoard(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.SevenDeuce = 40 })
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialDoubleBomb})
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

func TestSpecialSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*Settings)
	}{
		{"bomb too big", func(s *Settings) { s.BombPot = 21 * s.BigBlind }},
		{"scheduled bombs with bomb pots off", func(s *Settings) { s.SpecialEvery = 5 }},
		{"too many hands apart", func(s *Settings) { s.BombPot, s.SpecialEvery = 20, 101 }},
		{"too many orbits apart", func(s *Settings) { s.BombPot, s.SpecialEvery, s.SpecialUnit = 20, 21, UnitOrbits }},
		{"another unit", func(s *Settings) { s.SpecialUnit = "days" }},
		{"another special", func(s *Settings) { s.Special = "triple" }},
		{"scheduled double boards with bomb pots off", func(s *Settings) { s.Special, s.SpecialEvery = SpecialDoubleBomb, 5 }},
		{"double board alone", func(s *Settings) { s.BombPot, s.Special = 20, "double" }},
		{"two PLO5 boards for 9", func(s *Settings) {
			s.Seats, s.Game, s.BombPot, s.Special, s.SpecialEvery = 9, GameOmaha5, 20, SpecialDoubleBomb, 1
		}},
	} {
		set := DefaultSettings
		tt.edit(&set)
		if set.Validate() == nil {
			t.Errorf("%s: valid", tt.name)
		}
	}
	ok := DefaultSettings
	ok.Seats, ok.Game, ok.BombPot, ok.Special, ok.SpecialEvery = 8, GameOmaha5, 20, SpecialDoubleBomb, 1
	if err := ok.Validate(); err != nil {
		t.Fatalf("PLO5 double board for 8: %v", err)
	}

	h := bombTable(t, nil)
	set := h.s.Settings
	set.Special = SpecialDoubleBomb
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("double board bomb pots on")
	set.SpecialEvery, set.SpecialUnit = 2, UnitOrbits
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("a double board bomb pot every 2 orbits")
	if v := ViewFor(h.s, h.ids["a"], h.now); v.SpecialIn != 6 {
		t.Fatalf("next special in %d hands, want 2 orbits of 3", v.SpecialIn)
	}
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialBomb})
	set.BombPot, set.SpecialEvery = 0, 0
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("bomb pots to off, bomb pots only when called")
	if h.s.SpecialNext != "" {
		t.Fatal("bomb pot still next with bomb pots off")
	}
	h.refused(SpecialHand{Person: h.ids["a"], Kind: SpecialDoubleBomb})
}

func TestSpecialOrbits(t *testing.T) {
	// a double board bomb pot every orbit, three handed: every third hand
	h := bombTable(t, func(s *Settings) { s.Special, s.SpecialEvery, s.SpecialUnit = SpecialDoubleBomb, 1, UnitOrbits })
	h.ok(Start{Person: h.ids["a"]})
	var special []int
	for hand := 1; hand <= 7; hand++ {
		if h.s.Game.Bomb > 0 && h.s.Game.Double {
			special = append(special, hand)
		}
		for h.s.Game.InHand() {
			if h.s.Game.Betting {
				h.play(holdem.Fold{})
			} else {
				h.advance(time.Second)
			}
		}
		if hand < 7 {
			h.nextHand()
		}
	}
	if !slices.Equal(special, []int{3, 6}) {
		t.Fatalf("specials on hands %v", special)
	}
}

func TestHostSpecialWaitsForSchedule(t *testing.T) {
	h := bombTable(t, func(s *Settings) { s.SpecialEvery = 2 })
	h.ok(SpecialHand{Person: h.ids["a"], Kind: SpecialDoubleBomb})
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.Double || h.s.Game.Bomb != 50 {
		t.Fatalf("the host's double board wasn't dealt: double %v bomb %d", h.s.Game.Double, h.s.Game.Bomb)
	}
	for h.s.Game.InHand() {
		if h.s.Game.Betting {
			h.play(holdem.Fold{})
		} else {
			h.advance(time.Second)
		}
	}
	// the host's special didn't count, so the schedule's bomb comes after one normal hand
	if h.s.SpecialNext != "" || ViewFor(h.s, h.ids["a"], h.now).SpecialIn != 2 {
		t.Fatalf("next %q in %d", h.s.SpecialNext, ViewFor(h.s, h.ids["a"], h.now).SpecialIn)
	}
}
