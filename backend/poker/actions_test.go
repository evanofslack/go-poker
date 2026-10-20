package poker

import (
	"testing"
	"time"

	. "github.com/alexclewontin/riverboat/eval"
)

// newTestGame returns a game with the blinds these scenarios were written against
func newTestGame() *Game {
	g := NewGame()
	g.config.BigBlind = 25
	return g
}

func TestIntegration_Scenarios(t *testing.T) {

	t.Run("Scenario 1", func(t *testing.T) {
		g := newTestGame()

		pn_a := g.AddPlayer()
		g.AddPlayer()
		g.AddPlayer()

		err := Deal(g, pn_a, 0)

		if err != ErrIllegalAction {
			t.Error("Test failed - Deal must return ErrIllegalAction as 0 players are marked ready.")
		}
	})

	t.Run("Scenario 2", func(t *testing.T) {
		g := newTestGame()

		pn_a := g.AddPlayer()
		g.AddPlayer()
		g.AddPlayer()

		err := ToggleReady(g, pn_a, 0)

		if err != ErrIllegalAction {
			t.Error("Test failed - ToggleReady must return ErrIllegalAction as player 0 has not bought in.")
		}
	})

	t.Run("Scenario 3", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = Deal(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error dealing: %s", err)
		}
	})

	t.Run("Scenario 4", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = Deal(g, pn_b, 0)

		if err != ErrIllegalAction {
			t.Errorf("Test failed - must return ErrIllegalAction as pn_b is not the dealer")
		}
	})

	t.Run("Scenario 5", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = Deal(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error dealing: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {

			t.Errorf("Test failed - error betting: %s", err)
		}

		if g.players[pn_a].Bet != 25 {
			t.Errorf("Betting mechanic not working.")
		}
	})

	t.Run("Scenario 6 simple", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		// Preflop

		err = Deal(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error dealing: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_b, 15)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Flop

		err = Bet(g, pn_b, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Turn

		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		//River
		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

	})

	t.Run("Scenario 7 fold", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		// Preflop

		err = Deal(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error dealing: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_b, 15)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Flop

		err = Bet(g, pn_b, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Fold(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Turn
		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		//River

		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

	})

	t.Run("Scenario 8 reraise", func(t *testing.T) {
		var err error
		g := newTestGame()

		pn_a := g.AddPlayer()
		pn_b := g.AddPlayer()
		pn_c := g.AddPlayer()

		err = BuyIn(g, pn_a, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_b, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = BuyIn(g, pn_c, 100)

		if err != nil {
			t.Errorf("Test failed - Error buying in: %s", err)
		}

		err = ToggleReady(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		err = ToggleReady(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - Error marking ready: %s", err)
		}

		// Preflop

		err = Deal(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error dealing: %s", err)
		}

		err = Bet(g, pn_a, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_b, 15)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Flop

		err = Bet(g, pn_b, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Fold(g, pn_c, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 50)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_b, 25)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		// Turn

		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		//River

		err = Bet(g, pn_b, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

		err = Bet(g, pn_a, 0)

		if err != nil {
			t.Errorf("Test failed - error betting: %s", err)
		}

	})
	t.Run("Scenario 9 set username", func(t *testing.T) {
		g := newTestGame()

		pn_a := g.AddPlayer()
		g.AddPlayer()
		g.AddPlayer()

		username := "test"

		err := setUsername(g, pn_a, username)

		if err != nil {
			t.Error("Test failed - Could not set username.")
		}
		view := g.GeneratePlayerView(pn_a)

		if view.Players[0].Username != username {
			t.Error("Test failed - Could not set username.")
		}
	})

	t.Run("Scenario 10 set player seatIDs", func(t *testing.T) {
		g := newTestGame()

		pn_a := g.AddPlayer()
		setUsername(g, pn_a, "pn_a")
		err := setSeatID(g, pn_a, 2)
		if err != nil {
			t.Error("Test failed - setting valid position should not cause error")
		}

		pn_b := g.AddPlayer()
		setUsername(g, pn_b, "pn_b")
		err = setSeatID(g, pn_b, 5)
		if err != nil {
			t.Error("Test failed - setting valid position should not cause error")
		}

		pn_c := g.AddPlayer()
		setUsername(g, pn_c, "pn_c")
		err = setSeatID(g, pn_c, 9)
		if err != nil {
			t.Error("Test failed - setting valid position should not cause error")
		}

		pn_d := g.AddPlayer()
		setUsername(g, pn_d, "pn_d")
		err = setSeatID(g, pn_d, 1)
		if err != nil {
			t.Error("Test failed - setting valid position should not cause error")
		}
		// players are only sorted by seat at the next deal
		if g.players[3].Username != "pn_d" {
			t.Error("Test failed - setting a seat must not reorder players")
		}

		for i, p := range g.players {
			if p.Position != uint(i) {
				t.Error("Test failed - player positions are not correct")

			}
		}

		pn_e := g.AddPlayer()
		err = setSeatID(g, pn_e, 9)
		if err != ErrInvalidPosition {
			t.Error("Test failed - setting repeated position should raise error")
		}
	})
}

func readyPlayers(t *testing.T, g *Game, n int) {
	for i := 0; i < n; i++ {
		pn := g.AddPlayer()
		if err := BuyIn(g, pn, 100); err != nil {
			t.Fatal(err)
		}
		if err := SetSeatID(g, pn, uint(i+1)); err != nil {
			t.Fatal(err)
		}
		if err := ToggleReady(g, pn, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResetThenStart(t *testing.T) {
	g := newTestGame()
	readyPlayers(t, g, 3)
	g.dealerNum = 2
	g.Reset()
	readyPlayers(t, g, 2)
	if err := g.Start(); err != nil {
		t.Fatalf("start after reset: %s", err)
	}
}

func TestSeatZeroRejected(t *testing.T) {
	g := newTestGame()
	pn := g.AddPlayer()
	if err := SetSeatID(g, pn, 0); err != ErrInvalidPosition {
		t.Errorf("seat 0: got %v, want ErrInvalidPosition", err)
	}
}

func TestFoldBeforeDeal(t *testing.T) {
	g := newTestGame()
	readyPlayers(t, g, 2)
	if err := Fold(g, g.actionNum, 0); err != ErrIllegalAction {
		t.Errorf("fold before deal: got %v, want ErrIllegalAction", err)
	}
}

func TestDealerUnreadyWrapsAround(t *testing.T) {
	g := newTestGame()
	readyPlayers(t, g, 2)
	g.dealerNum = 1
	if err := ToggleReady(g, 1, 0); err != nil {
		t.Fatal(err)
	}
	if g.dealerNum != 0 {
		t.Errorf("dealer = %d, want 0", g.dealerNum)
	}
	if err := ToggleReady(g, 0, 0); err != nil {
		t.Fatal(err)
	}
}

// seatPlayers adds one ready player per stack, seated in order
func seatPlayers(t *testing.T, g *Game, stacks ...uint) {
	for i, s := range stacks {
		pn := g.AddPlayer()
		if err := BuyIn(g, pn, s); err != nil {
			t.Fatal(err)
		}
		if err := SetSeatID(g, pn, uint(i+1)); err != nil {
			t.Fatal(err)
		}
		if err := ToggleReady(g, pn, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func totalChips(g *Game) uint {
	var sum uint
	for _, p := range g.players {
		sum += p.Stack + p.TotalBet
	}
	return sum
}

func TestShortAllInCall(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 15, 100, 100)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	// p0 is utg facing 25 with only 15
	if err := Bet(g, 0, 15); err != nil {
		t.Fatalf("short all-in call: %s", err)
	}
	if !g.players[0].allIn() {
		t.Error("p0 should be all in")
	}
	if g.actionNum != 1 {
		t.Errorf("action = %d, want 1", g.actionNum)
	}
}

func TestShortAllInRaiseDoesNotReopen(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 100, 120, 120)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	// p0 (utg) shoves 100: a full raise, min raise is now 75
	if err := Bet(g, 0, 100); err != nil {
		t.Fatal(err)
	}
	if err := Bet(g, 1, 90); err != nil {
		t.Fatal(err)
	}
	// p2 (bb) shoves 95 more for 120 total: more than a call, less than a min raise
	if err := Bet(g, 2, 95); err != nil {
		t.Fatalf("short all-in raise: %s", err)
	}
	if g.minRaise != 75 {
		t.Errorf("minRaise = %d, want 75", g.minRaise)
	}
	if !g.players[1].Called {
		t.Error("short all-in must not reset Called")
	}
	if g.actionNum != 1 {
		t.Fatalf("action = %d, want 1 to call the extra 20", g.actionNum)
	}
	if err := Bet(g, 1, 50); err != ErrIllegalAction {
		t.Errorf("re-raise after short all-in: got %v, want ErrIllegalAction", err)
	}
	if err := Bet(g, 1, 20); err != nil {
		t.Fatal(err)
	}
	if g.getStage() != PreDeal || g.getBetting() {
		t.Errorf("hand should be over, stage %d betting %v", g.getStage(), g.getBetting())
	}
	if got := totalChips(g); got != 340 {
		t.Errorf("total chips = %d, want 340", got)
	}
}

func TestRunOutBoard(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 100, 50)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	// heads up: p0 is dealer and sb, acts first
	if err := Bet(g, 0, 90); err != nil {
		t.Fatal(err)
	}
	if err := Bet(g, 1, 25); err != nil {
		t.Fatal(err)
	}
	if g.getStage() != PreDeal || g.getBetting() {
		t.Fatalf("hand should be over, stage %d betting %v", g.getStage(), g.getBetting())
	}
	for i, c := range g.communityCards {
		if c == 0 {
			t.Errorf("community card %d not dealt", i)
		}
	}
	if got := totalChips(g); got != 150 {
		t.Errorf("total chips = %d, want 150", got)
	}
	if g.players[0].Stack < 50 {
		t.Errorf("p0 stack = %d, the 50 uncalled chips must be returned", g.players[0].Stack)
	}
	for i, p := range g.players {
		if p.In || p.Called {
			t.Errorf("player %d: In %v Called %v after hand", i, p.In, p.Called)
		}
		if p.Stack == 0 {
			if err := BuyIn(g, uint(i), 100); err != nil {
				t.Errorf("busted player %d can't rebuy: %s", i, err)
			}
		}
	}
}

func TestFoldResetsIn(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 100, 100)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	if err := Fold(g, 0, 0); err != nil {
		t.Fatal(err)
	}
	for i, p := range g.players {
		if p.In || p.Called {
			t.Errorf("player %d: In %v Called %v after hand", i, p.In, p.Called)
		}
	}
}

func TestSplitPotOddChip(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 100, 100, 100)
	g.setStageAndBetting(River, true)
	g.dealerNum = 0
	g.communityCards = []Card{
		MustParseCardString("AS"),
		MustParseCardString("KS"),
		MustParseCardString("QS"),
		MustParseCardString("JS"),
		MustParseCardString("TS"),
	}
	hole := [][2]Card{
		{MustParseCardString("2C"), MustParseCardString("3D")},
		{MustParseCardString("2D"), MustParseCardString("3C")},
		{MustParseCardString("4C"), MustParseCardString("5D")},
	}
	for i := range g.players {
		g.players[i].Stack = 75
		g.players[i].TotalBet = 25
		g.players[i].Cards = hole[i]
		g.players[i].In = i != 2
		g.players[i].Called = true
	}
	g.updateRoundInfo()

	// 75 split two ways: the odd chip goes to p1, first left of the dealer
	if g.players[0].Stack != 112 || g.players[1].Stack != 113 {
		t.Errorf("stacks = %d, %d, want 112, 113", g.players[0].Stack, g.players[1].Stack)
	}
	if got := totalChips(g); got != 300 {
		t.Errorf("total chips = %d, want 300", got)
	}
}

func TestNoReadyPlayersNoHang(t *testing.T) {
	g := newTestGame()
	g.AddPlayer()
	g.AddPlayer()
	done := make(chan struct{})
	go func() {
		g.resetForNextHand()
		g.updateBlindNums()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hung with no ready players")
	}
}

func TestStartSkipsUnready(t *testing.T) {
	g := newTestGame()
	idle := g.AddPlayer()
	if err := BuyIn(g, idle, 100); err != nil {
		t.Fatal(err)
	}
	pn := g.AddPlayer()
	if err := BuyIn(g, pn, 100); err != nil {
		t.Fatal(err)
	}
	if err := ToggleReady(g, pn, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.Start(); err != ErrStartGame {
		t.Fatalf("start with 1 ready: got %v, want ErrStartGame", err)
	}
	pn = g.AddPlayer()
	if err := BuyIn(g, pn, 100); err != nil {
		t.Fatal(err)
	}
	if err := ToggleReady(g, pn, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.Start(); err != nil {
		t.Fatalf("start with 2 ready: %s", err)
	}
	for _, num := range []uint{g.dealerNum, g.sbNum, g.bbNum, g.actionNum} {
		if !g.players[num].In {
			t.Errorf("player %d is dealer, a blind or to act but not in", num)
		}
	}
	if g.players[idle].In || g.players[idle].Cards[0] != 0 {
		t.Error("unready player was dealt in")
	}
}

func TestSeatMidHand(t *testing.T) {
	g := newTestGame()
	g.AddPlayer()
	g.AddPlayer()
	for pn, seat := range []uint{2, 4} {
		if err := BuyIn(g, uint(pn), 100); err != nil {
			t.Fatal(err)
		}
		if err := SetSeatID(g, uint(pn), seat); err != nil {
			t.Fatal(err)
		}
		if err := ToggleReady(g, uint(pn), 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}

	uuids := []string{g.players[0].UUID, g.players[1].UUID}
	action, dealer, sb, bb := g.actionNum, g.dealerNum, g.sbNum, g.bbNum

	// a new player sits in seat 1 mid-hand
	pn := g.AddPlayer()
	if err := BuyIn(g, pn, 100); err != nil {
		t.Fatal(err)
	}
	if err := ToggleReady(g, pn, 0); err != nil {
		t.Fatal(err)
	}
	if err := SetSeatID(g, pn, 1); err != nil {
		t.Fatal(err)
	}

	for i, u := range uuids {
		if g.players[i].UUID != u {
			t.Errorf("player %d moved mid-hand", i)
		}
	}
	if g.actionNum != action || g.dealerNum != dealer || g.sbNum != sb || g.bbNum != bb {
		t.Error("hand indexes changed mid-hand")
	}
	if g.players[pn].In {
		t.Error("new player should sit out the current hand")
	}

	// finish the hand and deal the next one
	if err := Fold(g, g.actionNum, 0); err != nil {
		t.Fatal(err)
	}
	dealerUUID := g.players[g.dealerNum].UUID
	if err := Deal(g, g.dealerNum, 0); err != nil {
		t.Fatal(err)
	}
	for i, p := range g.players {
		if p.Position != uint(i) {
			t.Errorf("player %d has position %d", i, p.Position)
		}
		if !p.In {
			t.Errorf("player %d (seat %d) not dealt in", i, p.SeatID)
		}
		if i > 0 && g.players[i-1].SeatID > p.SeatID {
			t.Error("players not sorted by seat")
		}
	}
	if g.players[g.dealerNum].UUID != dealerUUID {
		t.Error("dealer changed across the sort")
	}
}

func TestRemovePlayer(t *testing.T) {
	g := newTestGame()
	seatPlayers(t, g, 100, 100, 100, 100)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	if err := RemovePlayer(g, 3, 0); err != ErrIllegalAction {
		t.Fatalf("remove mid-hand: got %v, want ErrIllegalAction", err)
	}
	for g.getBetting() {
		if err := Fold(g, g.actionNum, 0); err != nil {
			t.Fatal(err)
		}
	}

	// dealer is now 1; removing 0 shifts everyone down
	if g.dealerNum != 1 {
		t.Fatalf("dealer = %d, want 1", g.dealerNum)
	}
	dealerUUID := g.players[1].UUID
	if err := RemovePlayer(g, 0, 0); err != nil {
		t.Fatal(err)
	}
	if len(g.players) != 3 || g.players[g.dealerNum].UUID != dealerUUID {
		t.Error("dealer moved after removing another player")
	}

	// removing the dealer points at the next player
	nextUUID := g.players[g.dealerNum+1].UUID
	if err := RemovePlayer(g, g.dealerNum, 0); err != nil {
		t.Fatal(err)
	}
	if len(g.players) != 2 || g.players[g.dealerNum].UUID != nextUUID {
		t.Error("dealer should move to the next player")
	}
	for i, p := range g.players {
		if p.Position != uint(i) {
			t.Errorf("player %d has position %d", i, p.Position)
		}
	}
	// heads up: dealer posts the small blind
	if g.sbNum != g.dealerNum || g.bbNum == g.dealerNum {
		t.Errorf("blinds not recomputed: dealer %d sb %d bb %d", g.dealerNum, g.sbNum, g.bbNum)
	}
	for i := range g.players {
		g.GeneratePlayerView(uint(i))
	}
	if err := Deal(g, g.dealerNum, 0); err != nil {
		t.Fatal(err)
	}
}
