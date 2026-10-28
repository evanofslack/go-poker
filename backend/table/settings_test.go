package table

import (
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

func TestValidSettings(t *testing.T) {
	ok := Settings{Seats: 8, SmallBlind: 10, BigBlind: 20, MinBuy: 1000, MaxBuy: 4000, TurnTime: 30000, NextHandDelay: 5000,
		DecisionTime: 5000, Straddle: StraddleOff, Game: GameHoldem, Limit: LimitNo, Deck: DeckFull,
		Special: SpecialBomb, SpecialUnit: UnitHands}
	tests := []struct {
		name  string
		edit  func(*Settings)
		valid bool
	}{
		{"defaults", func(*Settings) {}, true},
		{"heads up", func(s *Settings) { s.Seats = 2 }, true},
		{"nine seats", func(s *Settings) { s.Seats = 9 }, true},
		{"one seat", func(s *Settings) { s.Seats = 1 }, false},
		{"ten seats", func(s *Settings) { s.Seats = 10 }, false},
		{"equal blinds", func(s *Settings) { s.SmallBlind = 20 }, true},
		{"no small blind", func(s *Settings) { s.SmallBlind = 0 }, false},
		{"small above big", func(s *Settings) { s.SmallBlind = 21 }, false},
		{"huge big blind", func(s *Settings) { s.BigBlind = maxBigBlind + 1 }, false},
		{"min of one big blind", func(s *Settings) { s.MinBuy = 20 }, true},
		{"min under a big blind", func(s *Settings) { s.MinBuy = 19 }, false},
		{"min above max", func(s *Settings) { s.MinBuy = 4001 }, false},
		{"max at the cap", func(s *Settings) { s.MaxBuy = 20 * maxBuyInBB }, true},
		{"max over the cap", func(s *Settings) { s.MaxBuy = 20*maxBuyInBB + 1 }, false},
		{"no turn timer", func(s *Settings) { s.TurnTime = 0 }, true},
		{"turn too short", func(s *Settings) { s.TurnTime = minTurnTime - 1 }, false},
		{"turn too long", func(s *Settings) { s.TurnTime = maxTurnTime + 1 }, false},
		{"no hand delay", func(s *Settings) { s.NextHandDelay = 0 }, false},
		{"hand delay too long", func(s *Settings) { s.NextHandDelay = maxNextHandDelay + 1 }, false},
		{"ante of a big blind", func(s *Settings) { s.Ante = 20 }, true},
		{"ante over the big blind", func(s *Settings) { s.Ante = 21 }, false},
		{"7-2 at the cap", func(s *Settings) { s.SevenDeuce = 20 * maxSevenDeuceBB }, true},
		{"7-2 over the cap", func(s *Settings) { s.SevenDeuce = 20*maxSevenDeuceBB + 1 }, false},
		{"decision time too short", func(s *Settings) { s.DecisionTime = minDecisionTime - 1 }, false},
		{"decision time too long", func(s *Settings) { s.DecisionTime = maxDecisionTime + 1 }, false},
		{"forced straddles", func(s *Settings) { s.Straddle = StraddleForced }, true},
		{"no straddle setting", func(s *Settings) { s.Straddle = "" }, false},
		{"unknown straddle setting", func(s *Settings) { s.Straddle = "sometimes" }, false},
		{"pot limit hold'em", func(s *Settings) { s.Limit = LimitPot }, true},
		{"nine seats of omaha", func(s *Settings) { s.Game, s.Seats = GameOmaha, 9 }, true},
		{"nine seats of omaha 5", func(s *Settings) { s.Game, s.Seats = GameOmaha5, 9 }, true},
		{"no game", func(s *Settings) { s.Game = "" }, false},
		{"unknown game", func(s *Settings) { s.Game = "stud" }, false},
		{"unknown limit", func(s *Settings) { s.Limit = "fixed" }, false},
		{"7-2 in omaha", func(s *Settings) { s.Game, s.SevenDeuce = GameOmaha, 20 }, false},
	}
	for _, tt := range tests {
		s := ok
		tt.edit(&s)
		if err := s.Validate(); (err == nil) != tt.valid {
			t.Errorf("%s: Validate = %v, want valid %v", tt.name, err, tt.valid)
		}
	}
}

// blindBets are the small and big blinds posted this hand.
func (h *harness) blindBets() (uint, uint) {
	g := &h.s.Game
	return g.Player(g.SB).Bet, g.Player(g.BB).Bet
}

func TestUpdateSettingsBetweenHands(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)

	next := DefaultSettings
	next.SmallBlind, next.BigBlind, next.Ante = 25, 50, 5
	next.TurnTime = 60000
	next.Seats = 2
	h.refused(UpdateSettings{Person: h.ids["b"], Settings: next})
	bad := next
	bad.Ante = 51
	if err := h.refused(UpdateSettings{Person: h.ids["a"], Settings: bad}); !strings.Contains(err, "ante") {
		t.Fatalf("refused with %q", err)
	}
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: next})
	if set := h.s.Settings; set.Seats != DefaultSettings.Seats || set.BigBlind != 50 || h.s.Next != nil {
		t.Fatalf("settings %+v, next %+v", set, h.s.Next)
	}
	h.mustLog("a changed the blinds to 25/50, the ante to 5, the turn timer to 60s")

	h.ok(Start{Person: h.ids["a"]})
	if sb, bb := h.blindBets(); sb != 25 || bb != 50 {
		t.Fatalf("blinds %d/%d", sb, bb)
	}
	if got := h.s.Timers[TimerTurn]; !got.Equal(h.now.Add(time.Minute)) {
		t.Fatalf("turn ends at %v, want a minute from now", got)
	}
}

func TestUpdateSettingsMidHand(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	for i, name := range []string{"a", "b", "c"} {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	h.ok(Start{Person: h.ids["a"]})

	next := DefaultSettings
	next.SmallBlind, next.BigBlind = 25, 50
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: next})
	h.mustLog("a changed the blinds to 25/50, from the next hand")
	if h.s.Settings.BigBlind != 20 || h.s.Game.Config.BigBlind != 20 || ViewFor(h.s, h.ids["b"], h.now).Next == nil {
		t.Fatalf("settings %+v config %+v", h.s.Settings, h.s.Game.Config)
	}
	// changing it back drops the change, and a later one replaces it
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: DefaultSettings})
	if h.s.Next != nil {
		t.Fatalf("next %+v after undoing", h.s.Next)
	}
	h.mustLog("a undid the settings changes for the next hand")
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: next})
	next.SmallBlind, next.BigBlind = 20, 40
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: next})

	h.play(holdem.Fold{})
	h.play(holdem.Fold{})
	if h.s.Settings.BigBlind != 40 || h.s.Next != nil {
		t.Fatalf("after the hand: settings %+v next %+v", h.s.Settings, h.s.Next)
	}
	h.mustLog("settings changed: the blinds to 20/40")
	h.nextHand()
	if sb, bb := h.blindBets(); sb != 20 || bb != 40 {
		t.Fatalf("blinds %d/%d", sb, bb)
	}
}

func TestUpdateSettingsAfterEnd(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.ok(EndSession{Person: h.ids["a"]})
	h.refused(UpdateSettings{Person: h.ids["a"], Settings: DefaultSettings})
	h.refused(SetPassword{Person: h.ids["a"], Password: "pw"})
}

func TestBotRebuyKeepsToBuyInLimits(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.ok(AddBot{Person: h.ids["a"], Seat: 2, BuyIn: 1000, Name: "bot", Level: "easy"})
	next := DefaultSettings
	next.MinBuy = 2000
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: next})
	h.s.Game.Player(2).Stack = 0
	(&reducer{s: h.s, now: h.now}).rebuyBots()
	if got := h.s.Game.Player(2).Stack; got != 2000 {
		t.Fatalf("bot rebought for %d, want the new min of 2000", got)
	}
}

func TestSetPassword(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.join("b")
	h.refused(SetPassword{Person: h.ids["b"], Password: "pw"})
	h.ok(SetPassword{Person: h.ids["a"], Password: "pw"})
	if got := sentTo[NewPassword](h, h.ids["b"]); len(got) != 1 || got[0].Password != "pw" {
		t.Fatalf("b was sent %+v", got)
	}
	h.mustLog("a changed the password")
	if res := h.do(Join{ID: "id-c", Name: "c", Password: "old"}); res.Joined != nil {
		t.Fatal("joined with the wrong password")
	}
	if res := h.do(Join{ID: "id-c", Name: "c", Password: "pw"}); res.Joined == nil {
		t.Fatal("couldn't join with the new password")
	}

	h.ok(SetPassword{Person: h.ids["a"], Password: ""})
	h.mustLog("a removed the password")
	if ViewFor(h.s, h.ids["a"], h.now).HasPassword {
		t.Fatal("still has a password")
	}
	if res := h.do(Join{ID: "id-d", Name: "d"}); res.Joined == nil {
		t.Fatal("couldn't join without a password")
	}
}

func TestGameSettings(t *testing.T) {
	h := pair(t)
	set := h.s.Settings
	set.Game, set.Limit = GameOmaha, LimitPot
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	h.mustLog("the game to PLO")
	if g := h.s.Game.Config.Game; g.HoleCards() != 4 || !g.Pot {
		t.Fatalf("game %+v", g)
	}
	h.ok(Start{Person: h.ids["a"]})
	if v := ViewFor(h.s, h.ids["a"], h.now); len(v.Players[0].Cards) != 4 || len(v.Players[1].Cards) != 4 {
		t.Fatalf("dealt %v and %v", v.Players[0].Cards, v.Players[1].Cards)
	}
	if h.s.Current.Game != "PLO" {
		t.Fatalf("hand recorded as %q", h.s.Current.Game)
	}
	// mid-hand a change waits for the next hand
	set.Game, set.Limit = GameHoldem, LimitNo
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	if h.s.Game.Config.Game.HoleCards() != 4 {
		t.Fatal("game changed mid-hand")
	}
	h.play(holdem.Fold{})
	if h.s.Game.Config.Game.HoleCards() != 2 || GameName(h.s.Settings) != "NLH" {
		t.Fatalf("game after the hand %+v", h.s.Game.Config.Game)
	}
}

func TestRestoreBeforeGames(t *testing.T) {
	h := pair(t)
	h.s.Settings.Game, h.s.Settings.Limit, h.s.Settings.Deck = "", "", ""
	Restore(h.s, h.now)
	if h.s.Settings.Game != GameHoldem || h.s.Settings.Limit != LimitNo || h.s.Settings.Deck != DeckFull || h.s.Settings.Validate() != nil {
		t.Fatalf("restored as %q %q %q", h.s.Settings.Game, h.s.Settings.Limit, h.s.Settings.Deck)
	}
}
