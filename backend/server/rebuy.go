package server

import (
	"fmt"
	"slices"
	"time"
)

// buyRequest is a rebuy, or a returning player sitting back down, waiting for
// the host on a table that approves rebuys.
type buyRequest struct {
	Name   string    `json:"name"`
	Amount uint      `json:"amount"`
	Seat   uint      `json:"seat"` // the seat asked for, 0 to rebuy at their seat
	at     time.Time // when it was asked, for ordering
}

// needsApproval reports whether name's rebuys wait for the host. The host's
// own don't. Caller holds t.mu.
func needsApproval(t *table, name string) bool {
	return t.settings.ApproveRebuys && name != t.host
}

// requestBuy puts r in front of the host, replacing any earlier request from
// the same player. Caller holds t.mu.
func requestBuy(t *table, r buyRequest) {
	r.at = time.Now()
	t.requests[r.Name] = r
	if r.Seat == 0 {
		t.send(createNewLog(fmt.Sprintf("%s asks to rebuy for %d", r.Name, r.Amount)))
	} else {
		t.send(createNewLog(fmt.Sprintf("%s asks to sit back down with %d", r.Name, r.Amount)))
	}
	broadcastGame(t)
}

// buyRequests lists the requests waiting for the host, oldest first. Caller
// holds t.mu.
func (t *table) buyRequests() []buyRequest {
	rs := make([]buyRequest, 0, len(t.requests))
	for _, r := range t.requests {
		rs = append(rs, r)
	}
	slices.SortFunc(rs, func(a, b buyRequest) int { return a.at.Compare(b.at) })
	return rs
}

// userByName finds the connected client called name. Caller holds t.mu.
func (t *table) userByName(name string) *Client {
	for u := range t.users {
		if u.username == name {
			return u
		}
	}
	return nil
}

// handleApproveBuy carries out name's request, checking it again first. One
// that no longer fits is dropped; a rebuy mid-hand waits for the hand to end.
func handleApproveBuy(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	r, ok := t.requests[name]
	if !ok {
		sendError(c, fmt.Sprintf("%s hasn't asked to buy in", name))
		return
	}
	drop := func(why string) {
		delete(t.requests, name)
		sendError(c, fmt.Sprintf("%s, so their request was dropped", why))
		broadcastGame(t)
	}

	if r.Seat == 0 {
		pn, ok := playerByName(t, name)
		if !ok {
			drop(fmt.Sprintf("%s doesn't have a seat", name))
			return
		}
		if t.game.GenerateOmniView().Players[pn].In {
			sendError(c, "wait for the hand to end")
			return
		}
		if msg := checkRebuy(t, pn, r.Amount); msg != "" {
			drop(fmt.Sprintf("%s's rebuy isn't allowed anymore", name))
			return
		}
		delete(t.requests, name)
		t.send(createNewLog(fmt.Sprintf("%s approved %s's rebuy", c.username, name)))
		applyRebuy(t, pn, r.Amount)
		return
	}

	u := t.userByName(name)
	switch {
	case u == nil:
		drop(fmt.Sprintf("%s isn't at the table", name))
		return
	case isSeatedUUID(t, u.id()): // not isSeated, u.table is u's to touch
		drop(fmt.Sprintf("%s already has a seat", name))
		return
	}
	if err := checkSeat(t, r.Seat, r.Amount); err != nil {
		drop(fmt.Sprintf("seat %d isn't free anymore", r.Seat))
		return
	}
	delete(t.requests, name)
	t.send(createNewLog(fmt.Sprintf("%s approved %s's buy-in", c.username, name)))
	seatClient(t, u, r.Seat, r.Amount)
}

// handleDenyBuy turns down name's request.
func handleDenyBuy(c *Client, name string) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isHost(c) {
		return
	}
	if _, ok := t.requests[name]; !ok {
		sendError(c, fmt.Sprintf("%s hasn't asked to buy in", name))
		return
	}
	delete(t.requests, name)
	t.send(createNewLog(fmt.Sprintf("%s turned down %s's buy-in", c.username, name)))
	broadcastGame(t)
}
