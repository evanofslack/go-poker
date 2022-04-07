package poker

import (
	"testing"
)

func TestIntegration_Scenarios(t *testing.T) {

	t.Run("Scenario 1", func(t *testing.T) {
		g := NewGame()

		pn_a := g.AddPlayer()
		g.AddPlayer()
		g.AddPlayer()

		err := Deal(g, pn_a, 0)

		if err != ErrIllegalAction {
			t.Error("Test failed - Deal must return ErrIllegalAction as 0 players are marked ready.")
		}
	})

	t.Run("Scenario 2", func(t *testing.T) {
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		g := NewGame()

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
		if g.players[0].Username != "pn_d" {
			t.Error("Test failed - player order does not match position")
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
