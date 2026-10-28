package table

import (
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

func TestAddTime(t *testing.T) {
	h := pair(t)
	if c := h.s.Seats[1].TimeCards; c != DefaultSettings.TimeBank {
		t.Fatalf("sat down with %d cards", c)
	}
	if msg := h.refused(AddTime{Person: h.ids["a"]}); msg != "you can only add time on your turn" {
		t.Fatalf("between hands: %q", msg)
	}
	h.ok(Start{Person: h.ids["a"]})
	if msg := h.refused(AddTime{Person: h.ids["b"]}); msg != "you can only add time on your turn" {
		t.Fatalf("not b's turn: %q", msg)
	}
	due := h.s.Timers[TimerTurn]
	h.ok(AddTime{Person: h.ids["a"]})
	if got := h.s.Timers[TimerTurn]; !got.Equal(due.Add(20 * time.Second)) {
		t.Fatalf("turn ends %v, want %v", got, due.Add(20*time.Second))
	}
	if c := h.s.Seats[1].TimeCards; c != 1 {
		t.Fatalf("a has %d cards", c)
	}
	h.mustLog("a adds 20s")
	if v := ViewFor(h.s, h.ids["b"], h.now); v.Players[0].TimeCards != 1 || v.TurnEndsIn != 50000 {
		t.Fatalf("b sees %d cards, %dms", v.Players[0].TimeCards, v.TurnEndsIn)
	}
	h.ok(AddTime{Person: h.ids["a"]})
	if msg := h.refused(AddTime{Person: h.ids["a"]}); msg != "you have no time cards left" {
		t.Fatalf("out of cards: %q", msg)
	}
	// moving on resets the turn timer to the usual
	h.play(holdem.Call{})
	if v := ViewFor(h.s, h.ids["b"], h.now); v.TurnEndsIn != 30000 {
		t.Fatalf("b's turn %dms", v.TurnEndsIn)
	}
}

func TestAddTimeNeedsTimer(t *testing.T) {
	set := DefaultSettings
	set.TurnTime = 0
	h := newHarness(t, set, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	h.ok(Start{Person: h.ids["a"]})
	if msg := h.refused(AddTime{Person: h.ids["a"]}); msg != "you can only add time on your turn" {
		t.Fatalf("no turn timer: %q", msg)
	}
}

func TestTimeBankOnTimeout(t *testing.T) {
	h := pair(t)
	h.ok(Start{Person: h.ids["a"]})
	// a's two cards go before the timer acts for them
	h.advance(30 * time.Second)
	if !h.s.Game.Betting || h.toAct() != "a" || h.s.Seats[1].TimeCards != 1 {
		t.Fatalf("first card not spent: %s, %d cards", h, h.s.Seats[1].TimeCards)
	}
	h.mustLog("a's time bank adds 20s")
	h.advance(20 * time.Second)
	if h.s.Seats[1].TimeCards != 0 || h.s.Game.Hands != 1 || !h.s.Game.InHand() {
		t.Fatalf("second card not spent: %s", h)
	}
	h.advance(20 * time.Second)
	if h.s.Game.InHand() && h.s.Game.Hands == 1 {
		t.Fatalf("a should have been folded: %s", h)
	}
}

func TestTimeBankDisconnected(t *testing.T) {
	h := pair(t)
	h.ok(Start{Person: h.ids["a"]})
	h.ok(Disconnect{Person: h.ids["a"]})
	h.advance(30 * time.Second)
	if h.s.Seats[1].TimeCards != 2 || len(h.s.History) != 1 {
		t.Fatalf("spent a disconnected player's card: %d cards, %d hands", h.s.Seats[1].TimeCards, len(h.s.History))
	}
}

func TestEarnTime(t *testing.T) {
	set := DefaultSettings
	set.TimeEvery, set.TimeMax = 2, 3
	h := newHarness(t, set, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	h.ok(Start{Person: h.ids["a"]})
	for hand := 1; hand <= 6; hand++ {
		h.play(holdem.Fold{})
		want := min(set.TimeBank+uint(hand)/2, 3)
		if c := h.s.Seats[1].TimeCards; c != want {
			t.Fatalf("after hand %d a has %d cards, want %d", hand, c, want)
		}
		h.advance(time.Duration(set.NextHandDelay) * time.Millisecond)
	}
	// leaving the seat loses them
	h.ok(LeaveSeat{Person: h.ids["a"]})
	h.advance(time.Minute)
	h.sit("a", 3, 1000)
	if c := h.s.Seats[3].TimeCards; c != set.TimeBank {
		t.Fatalf("sat back down with %d cards", c)
	}
}

func TestBotsHoldNoTime(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("host")
	h.ok(AddBot{Person: h.ids["host"], Seat: 1, Level: "easy", BuyIn: 1000})
	if c := h.s.Seats[1].TimeCards; c != 0 {
		t.Fatalf("bot has %d cards", c)
	}
}

func TestTimeBankSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*Settings)
	}{
		{"too many cards", func(s *Settings) { s.TimeMax = 11 }},
		{"start over the cap", func(s *Settings) { s.TimeBank = 6 }},
		{"card too short", func(s *Settings) { s.TimeCard = 1000 }},
		{"card too long", func(s *Settings) { s.TimeCard = 90_000 }},
		{"earn too slow", func(s *Settings) { s.TimeEvery = 500 }},
	} {
		set := DefaultSettings
		tt.edit(&set)
		if set.Validate() == nil {
			t.Errorf("%s: valid", tt.name)
		}
	}
	off := DefaultSettings
	off.TimeMax, off.TimeBank, off.TimeCard = 0, 0, 0
	if err := off.Validate(); err != nil {
		t.Fatalf("off: %v", err)
	}

	h := pair(t)
	set := DefaultSettings
	set.TimeMax, set.TimeBank = 1, 1
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("the time bank to 1 card of 20s, one more every 20 hands up to 1")
	if c := h.s.Seats[1].TimeCards; c != 1 {
		t.Fatalf("a kept %d cards over the new cap", c)
	}
	set.TimeMax, set.TimeBank = 0, 0
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("the time bank off")
}
