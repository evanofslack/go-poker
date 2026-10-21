package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/evanofslack/go-poker/poker"
)

// hostActions handle the host actions that only name a player.
var hostActions = map[string]func(*Client, string){
	actionMakeHost:        handleMakeHost,
	actionSetBanker:       handleSetBanker,
	actionMoveToSpectator: handleMoveToSpectator,
	actionRemovePlayer:    handleRemovePlayer,
	actionSitOutPlayer:    handleSitOutPlayer,
	actionApproveBuy:      handleApproveBuy,
	actionDenyBuy:         handleDenyBuy,
}

// isHost reports whether c is the host, and tells c if it isn't. Caller holds
// c.table.mu.
func isHost(c *Client) bool {
	if c.username != c.table.host {
		sendError(c, "only the host can do that")
		return false
	}
	return true
}

// handleMakeHost hands the host role to name, who must be at the table.
func handleMakeHost(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if name == c.username {
		sendError(c, "you're already the host")
		return
	}
	if !t.connected(name) {
		sendError(c, fmt.Sprintf("%s isn't at the table", name))
		return
	}
	t.host = name
	t.log(fmt.Sprintf("%s made %s the host", c.username, name))
	broadcastGame(t)
}

// handleSetBanker makes name the banker everyone settles up with. They must be
// at the table or in the ledger.
func handleSetBanker(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if !t.connected(name) && !t.inLedger(name) {
		sendError(c, fmt.Sprintf("%s isn't at the table", name))
		return
	}
	t.banker = name
	t.log(fmt.Sprintf("%s made %s the banker", c.username, name))
	broadcastGame(t)
}

// handleMoveToSpectator gives up name's seat, cashing them out, and keeps them
// at the table watching.
func handleMoveToSpectator(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	pn, ok := playerByName(t, name)
	if !ok {
		sendError(c, fmt.Sprintf("%s doesn't have a seat", name))
		return
	}
	t.log(fmt.Sprintf("%s moved %s to watching", c.username, name))
	unseat(t, t.game.GenerateOmniView().Players[pn].UUID)
	broadcastGame(t)
}

// handleRemovePlayer gives up name's seat and tells their client to leave the
// table. There are no bans: they can join again.
func handleRemovePlayer(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if name == c.username {
		sendError(c, "you can't remove yourself")
		return
	}
	pn, seated := playerByName(t, name)
	if !seated && !t.connected(name) {
		sendError(c, fmt.Sprintf("%s isn't at the table", name))
		return
	}
	t.log(fmt.Sprintf("%s removed %s from the table", c.username, name))
	if seated {
		unseat(t, t.game.GenerateOmniView().Players[pn].UUID)
	}
	for u := range t.users {
		if u.username == name {
			sendErrorCode(u, codeRemoved, "the host removed you from the table")
		}
	}
	broadcastGame(t)
}

// handleSitOutPlayer sits name out: right away between hands, else once the
// hand ends. Only they can sit back in.
func handleSitOutPlayer(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	pn, ok := playerByName(t, name)
	if !ok {
		sendError(c, fmt.Sprintf("%s doesn't have a seat", name))
		return
	}
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	switch {
	case !p.Ready || t.sitOutNext[p.UUID]:
		sendError(c, fmt.Sprintf("%s is already sitting out", name))
		return
	case view.Betting:
		// toggling mid-hand could move the dealer, so wait for the hand to end
		t.sitOutNext[p.UUID] = true
		t.log(fmt.Sprintf("%s will sit %s out after this hand", c.username, name))
	default:
		if err := poker.ToggleReady(t.game, pn, 0); err != nil {
			sendError(c, fmt.Sprintf("can't sit %s out right now", name))
			return
		}
		t.log(fmt.Sprintf("%s sat %s out", c.username, name))
	}
	broadcastGame(t)
}

// handleAdjustChips adds amount to name's stack, or takes it away when
// negative, between hands. The ledger counts it as a buy-in or cash-out.
func handleAdjustChips(c *Client, name string, amount int64, reason string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	pn, ok := playerByName(t, name)
	if !ok {
		sendError(c, fmt.Sprintf("%s doesn't have a seat", name))
		return
	}
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	if p.In {
		sendError(c, "wait for the hand to end")
		return
	}
	note := ""
	if reason != "" {
		note = fmt.Sprintf(" (%s)", reason)
	}

	if amount > 0 {
		add := uint(amount)
		if err := poker.BuyIn(t.game, pn, add); err != nil {
			sendError(c, fmt.Sprintf("can't take %s's stack above %d", name, view.Config.MaxBuy))
			return
		}
		// chips deal a busted player back in, like a rebuy
		if !p.Ready && p.Stack == 0 {
			if err := poker.ToggleReady(t.game, pn, 0); err != nil {
				slog.Default().Warn("Toggle ready after host add", "error", err)
			}
		}
		t.record(name, entryAdd, add, c.username, reason)
		t.log(fmt.Sprintf("%s added %d to %s's stack%s", c.username, add, name, note))
		if view.Running && view.Stage == poker.PreDeal {
			t.scheduleDeal()
		}
	} else {
		take := uint(-amount)
		if err := poker.RemoveChips(t.game, pn, take); err != nil {
			sendError(c, fmt.Sprintf("%s only has %d", name, p.Stack))
			return
		}
		// a player left with nothing sits out until they rebuy
		if p.Ready && p.Stack == take {
			if err := poker.ToggleReady(t.game, pn, 0); err != nil {
				slog.Default().Warn("Toggle ready after host remove", "error", err)
			}
		}
		t.record(name, entryRemove, take, c.username, reason)
		t.log(fmt.Sprintf("%s took %d from %s's stack%s", c.username, take, name, note))
	}
	broadcastGame(t)
}

// handlePause stops dealing new hands, or starts again. Mid-hand, the hand
// plays out first.
func handlePause(c *Client, pause bool) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if t.paused && pause {
		sendError(c, "the game is already paused")
		return
	}
	if !t.paused && !pause {
		sendError(c, "the game isn't paused")
		return
	}
	t.paused = pause
	view := t.game.GenerateOmniView()
	switch {
	case pause && view.Stage != poker.PreDeal:
		t.log(fmt.Sprintf("%s paused the game after this hand", c.username))
	case pause:
		t.log(fmt.Sprintf("%s paused the game", c.username))
	default:
		t.log(fmt.Sprintf("%s resumed the game", c.username))
		if view.Running && view.Stage == poker.PreDeal {
			t.scheduleDeal()
		}
	}
	broadcastGame(t)
}

// handleEndSession ends the session: right away between hands, else once
// the hand ends.
func handleEndSession(c *Client) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if t.ending {
		sendError(c, "the session already ends after this hand")
		return
	}
	if t.game.GenerateOmniView().Stage != poker.PreDeal {
		t.ending = true
		t.log(fmt.Sprintf("%s will end the session after this hand", c.username))
		broadcastGame(t)
		return
	}
	t.log(fmt.Sprintf("%s ended the session", c.username))
	endSession(t)
	broadcastGame(t)
}

// endSession stops dealing for good, so the ledger is final, and closes the
// table after endedTableTTL. Only works between hands. Caller holds t.mu.
func endSession(t *table) {
	t.ending = false
	t.endedAt = time.Now()
	clear(t.requests)
	if t.dealTimer != nil {
		t.dealTimer.Stop()
	}
	t.turnSeq++ // cancel the turn timer
	t.log("the session has ended, the ledger is final")
	time.AfterFunc(endedTableTTL, func() { t.hub.closeTable(t) })
}
