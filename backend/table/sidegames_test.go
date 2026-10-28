package table

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

func hole(s string) []holdem.Card {
	f := strings.Fields(s)
	return []holdem.Card{eval.MustParseCardString(f[0]), eval.MustParseCardString(f[1])}
}

func board(s string) []holdem.Card {
	var cards []holdem.Card
	for _, f := range strings.Fields(s) {
		cards = append(cards, eval.MustParseCardString(f))
	}
	return cards
}

// sevenDeuceTable seats a in 1 and b in 2, playing 7-2 for 40.
func sevenDeuceTable(t *testing.T) *harness {
	settings := DefaultSettings
	settings.SevenDeuce = 40
	h := newHarness(t, settings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	return h
}

// checkDown checks every street to the river, then sets the board before
// the last checks so the showdown is known.
func (h *harness) checkDown(river string) {
	h.t.Helper()
	for h.s.Game.Street != holdem.River || !h.s.Game.Betting {
		if h.s.Game.Betting {
			h.play(holdem.Check{})
		} else {
			h.advance(time.Second)
		}
	}
	h.s.Game.Board = board(river)
	h.s.Game.Boards = [][]holdem.Card{h.s.Game.Board}
	for h.s.Game.InHand() {
		h.play(holdem.Check{})
	}
}

func TestSevenDeuceFoldWin(t *testing.T) {
	h := sevenDeuceTable(t)
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(2).Cards = hole("7c 2d")
	h.play(holdem.Fold{}) // a, in the small blind

	if got := ViewFor(h.s, h.ids["b"], h.now).ShowClaims; !slices.Equal(got, []string{"7-2"}) {
		t.Fatalf("b's claims %v", got)
	}
	if h.stack("a") != 990 {
		t.Fatal("nobody pays before b shows")
	}
	h.ok(Show{Person: h.ids["b"]})
	if h.stack("a") != 950 || h.stack("b") != 1050 {
		t.Fatalf("stacks a %d b %d", h.stack("a"), h.stack("b"))
	}
	h.mustLog("b wins the 7-2: 40 from 1 player")
	rec := h.s.History[0]
	if len(rec.Payments) != 1 || rec.Players[0].Net != -50 || rec.Players[1].Net != 50 || h.s.Totals["b"].Net != 50 {
		t.Fatalf("record %+v totals %+v", rec, h.s.Totals["b"])
	}
	if got := ViewFor(h.s, h.ids["b"], h.now).ShowClaims; len(got) != 0 {
		t.Fatalf("already claimed: %v", got)
	}
}

func TestSevenDeuceShowdown(t *testing.T) {
	h := sevenDeuceTable(t)
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(1).Cards = hole("Ah Ad")
	h.s.Game.Player(2).Cards = hole("7c 2d")
	h.play(holdem.Call{})
	h.checkDown("7h 7s 2s Kc Qd") // b fills up

	// b showed to win, so is paid at once
	if h.stack("a") != 940 || h.stack("b") != 1060 {
		t.Fatalf("stacks a %d b %d", h.stack("a"), h.stack("b"))
	}
	if rec := h.s.History[0]; rec.Players[1].Net != 60 || len(rec.Payments) != 1 {
		t.Fatalf("record %+v", rec)
	}
}

func TestSevenDeuceSidePotDoesNotCount(t *testing.T) {
	settings := DefaultSettings
	settings.SevenDeuce = 40
	settings.MinBuy = 100
	h := newHarness(t, settings, "")
	for i, name := range []string{"a", "b", "c"} {
		h.join(name)
		h.sit(name, i+1, []uint{100, 1000, 1000}[i])
	}
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(1).Cards = hole("7d Kh")
	h.s.Game.Player(2).Cards = hole("7c 2d")
	h.s.Game.Player(3).Cards = hole("3c 4d")
	h.play(holdem.Raise{To: 100}) // a, all in
	h.play(holdem.Call{})
	h.play(holdem.Call{})
	h.advance(time.Second)
	h.play(holdem.Raise{To: 100}) // b bets the flop for a side pot
	h.play(holdem.Call{})
	h.checkDown("7h 7s 2s Kc Qd") // a fills up with kings and takes the main pot

	if pots := h.s.History[0].Pots; len(pots) != 2 || pots[0].Winners[0] != "a" || pots[1].Winners[0] != "b" {
		t.Fatalf("pots %+v", pots)
	}
	if len(h.s.History[0].Payments) != 0 || h.logged("7-2") {
		t.Fatal("a side pot doesn't win the 7-2")
	}
}

func TestSevenDeuceShortPayer(t *testing.T) {
	h := sevenDeuceTable(t)
	h.s.Game.Player(1).Stack = 25
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(2).Cards = hole("2s 7s") // suited counts
	h.play(holdem.Fold{})
	h.ok(Show{Person: h.ids["b"]})
	if a := h.s.Game.Player(1); a.Stack != 0 || !a.SittingOut || h.stack("b") != 1025 {
		t.Fatalf("a %+v, b %d", a, h.stack("b"))
	}
}

func TestSevenDeuceBotShows(t *testing.T) {
	settings := DefaultSettings
	settings.SevenDeuce = 40
	h := newHarness(t, settings, "")
	h.join("a")
	h.sit("a", 1, 1000)
	h.ok(AddBot{Person: h.ids["a"], Seat: 2, BuyIn: 1000, Name: "bot", Level: "easy"})
	h.ok(Start{Person: h.ids["a"]})
	h.s.Game.Player(2).Cards = hole("7c 2d")
	h.play(holdem.Fold{})
	if !h.s.Game.Player(2).Shown || h.stack("bot") != 1050 {
		t.Fatalf("the bot should show to claim: %s", h)
	}
}
