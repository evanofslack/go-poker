package table

import (
	"testing"

	"github.com/evanofslack/go-poker/holdem"
)

// shortTable is sevenDeuceTable on a short deck.
func shortTable(t *testing.T) *harness {
	set := DefaultSettings
	set.Deck, set.SevenDeuce = DeckShort, 40
	h := newHarness(t, set, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	return h
}

func TestShortDeckTable(t *testing.T) {
	h := shortTable(t)
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.Config.Game.Short || len(h.s.Game.Deck) != 36-4 {
		t.Fatalf("config %+v, %d cards left", h.s.Game.Config.Game, len(h.s.Game.Deck))
	}
	for _, p := range h.s.Game.Players {
		for _, c := range p.Cards {
			if r := c.String()[0]; r >= '2' && r <= '5' {
				t.Fatalf("dealt %v", c)
			}
		}
	}
	h.play(holdem.Fold{})
	if g := h.s.History[0].Game; g != "6+ NLH" {
		t.Fatalf("recorded %q", g)
	}
}

func TestJackSix(t *testing.T) {
	h := shortTable(t)
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(2).Cards = hole("Jc 6d")
	h.play(holdem.Fold{})
	if got := ViewFor(h.s, h.ids["b"], h.now).ShowClaims; len(got) != 1 || got[0] != "J-6" {
		t.Fatalf("b's claims %v", got)
	}
	h.ok(Show{Person: h.ids["b"]})
	h.mustLog("b wins the J-6: 40 from 1 player")
	if p := h.s.History[0].Payments; len(p) != 1 || p[0].Kind != KindJackSix {
		t.Fatalf("payments %+v", p)
	}
}

func TestSevenDeuceNotOnShortDeck(t *testing.T) {
	h := shortTable(t)
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(2).Cards = hole("7c 2d") // can't be dealt, but it isn't the worst hand here
	h.play(holdem.Fold{})
	if got := ViewFor(h.s, h.ids["b"], h.now).ShowClaims; len(got) != 0 {
		t.Fatalf("b's claims %v", got)
	}
}

func TestWorstHand(t *testing.T) {
	for _, tt := range []struct {
		g     holdem.Game
		cards string
		want  bool
	}{
		{holdem.Game{}, "7c 2d", true},
		{holdem.Game{}, "2d 7c", true},
		{holdem.Game{}, "Jc 6d", false},
		{holdem.Game{Short: true}, "6d Jc", true},
		{holdem.Game{Short: true}, "7c 2d", false},
		{holdem.Game{Hole: 4}, "7c 2d 3h 4s", false},
	} {
		if got := isWorstHand(tt.g, board(tt.cards)); got != tt.want {
			t.Errorf("%+v %s: %v", tt.g, tt.cards, got)
		}
	}
}

func TestShortDeckSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*Settings)
		ok   bool
	}{
		{"hold'em for 9", func(s *Settings) { s.Seats = 9 }, true},
		{"PLO for 7", func(s *Settings) { s.Seats, s.Game = 7, GameOmaha }, true},
		{"PLO for 8", func(s *Settings) { s.Seats, s.Game = 8, GameOmaha }, false},
		{"PLO5 for 6", func(s *Settings) { s.Seats, s.Game = 6, GameOmaha5 }, true},
		{"PLO5 for 7", func(s *Settings) { s.Seats, s.Game = 7, GameOmaha5 }, false},
		{"hold'em double board for 9", func(s *Settings) { s.Seats, s.DoubleBoard = 9, true }, true},
		{"PLO double board for 7", func(s *Settings) { s.Seats, s.Game, s.DoubleBoard = 7, GameOmaha, true }, false},
		{"another deck", func(s *Settings) { s.Deck = "tiny" }, false},
	} {
		set := DefaultSettings
		set.Deck = DeckShort
		tt.edit(&set)
		if err := set.Validate(); (err == nil) != tt.ok {
			t.Errorf("%s: %v", tt.name, err)
		}
	}

	h := pair(t)
	set := DefaultSettings
	set.Deck, set.SevenDeuce = DeckShort, 40
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("the game to 6+ NLH, the J-6 bounty to 40")
}
