package server

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/evanofslack/go-poker/bot"
	"github.com/evanofslack/go-poker/poker"
)

// botSeat is how a bot plays and what it buys back in for.
type botSeat struct {
	level string
	buyIn uint
}

// botDelay is how long a bot thinks before moving, a var so tests can shorten it.
var botDelay = func() time.Duration {
	return time.Second + rand.N(time.Second)
}

// handleAddBot has the host sit a bot down in an empty seat.
func handleAddBot(c *Client, seatID, buyIn uint, name, level string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.host != c {
		sendError(c, "only the host can add bots")
		return
	}
	if t.nameTaken(name) {
		sendError(c, "name taken")
		return
	}
	if err := checkSeat(t, seatID, buyIn); err != nil {
		sendError(c, err.Error())
		return
	}
	uuid := seatPlayer(t, name, seatID, buyIn)
	t.bots[uuid] = botSeat{level: level, buyIn: buyIn}
	broadcastGame(t)
}

// handleRemoveBot has the host take a bot out of the game. Mid-hand it's
// folded on its turn and removed after the hand, like a leaving player.
func handleRemoveBot(c *Client, seatID uint) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.host != c {
		sendError(c, "only the host can remove bots")
		return
	}
	for _, p := range t.game.GenerateOmniView().Players {
		if _, ok := t.bots[p.UUID]; ok && p.SeatID == seatID {
			leaveSeat(t, p.UUID)
			broadcastGame(t)
			return
		}
	}
	sendError(c, "no bot in that seat")
}

// scheduleBot has the player to act move after a pause, if they're a bot.
// seq is the turn, so the move is dropped if the turn has moved on. Caller
// holds t.mu.
func (t *table) scheduleBot(seq int) {
	view := t.game.GenerateOmniView()
	if !view.Betting {
		return
	}
	if _, ok := t.bots[view.Players[view.ActionNum].UUID]; ok {
		time.AfterFunc(botDelay(), func() { botMove(t, seq) })
	}
}

// botMove makes the bot to act's move, unless the turn has moved on.
func botMove(t *table, seq int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case <-t.done:
		return
	default:
	}
	view := t.game.GenerateOmniView()
	if seq != t.turnSeq || !view.Betting {
		return
	}
	pn := view.ActionNum
	seat, ok := t.bots[view.Players[pn].UUID]
	if !ok {
		return
	}

	// the bot sees only what its player would
	a := bot.Decide(seat.level, t.game.GeneratePlayerView(pn), pn, t.rng)
	var err error
	switch a.Kind {
	case bot.Raise:
		err = raise(t, pn, a.Amount)
	case bot.Call:
		err = call(t, pn)
	case bot.Check:
		err = check(t, pn)
	default:
		err = fold(t, pn)
	}
	if err == nil {
		return
	}
	slog.Default().Warn("Bot move", "error", err, "move", a)
	if check(t, pn) != nil && fold(t, pn) != nil {
		slog.Default().Warn("Bot fallback", "error", err)
	}
}

// rebuyBots buys busted bots back in for their buy-in. Only works between
// hands. Caller holds t.mu.
func rebuyBots(t *table) {
	for pn, p := range t.game.GenerateOmniView().Players {
		seat, ok := t.bots[p.UUID]
		if !ok || p.Stack > 0 || t.leaving[p.UUID] {
			continue
		}
		if err := poker.BuyIn(t.game, uint(pn), seat.buyIn); err != nil {
			slog.Default().Warn("Bot rebuy", "error", err)
			continue
		}
		if !p.Ready {
			if err := poker.ToggleReady(t.game, uint(pn), 0); err != nil {
				slog.Default().Warn("Toggle ready after bot rebuy", "error", err)
			}
		}
		t.send(createNewLog(fmt.Sprintf("%s rebuys for %d", p.Username, seat.buyIn)))
	}
}

// botIDs are the uuids of the bots, sorted. Caller holds t.mu.
func (t *table) botIDs() []string {
	ids := make([]string, 0, len(t.bots))
	for id := range t.bots {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
