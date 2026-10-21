package server

import (
	"net/http/httptest"
	"testing"

	"github.com/evanofslack/go-poker/poker"
	"github.com/gorilla/websocket"
)

const approveSettings = `{"seats":4,"sb":5,"bb":10,"minBuy":100,"maxBuy":10000,"turnTime":0,"nextHandDelay":2000,"approveRebuys":true}`

// approvalPair creates a table that approves rebuys, with host a in seat 1
// and b in seat 2, without starting.
func approvalPair(t *testing.T, srv *httptest.Server) (a, b *websocket.Conn, aRecv, bRecv received) {
	t.Helper()
	a, aRecv = dialRecv(t, srv)
	send(t, a, createWithSettings("t", "a", approveSettings), seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	b, bRecv = dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	return a, b, aRecv, bRecv
}

func TestRebuyWaitsForHost(t *testing.T) {
	a, b, aRecv, _ := approvalPair(t, newTestServer(t))
	send(t, b, `{"action":"rebuy","amount":100}`)
	waitLog(t, aRecv, "b asks to rebuy for 100")
	send(t, a, `{"action":"deny-buy-in","name":"b"}`)
	waitLog(t, aRecv, "a turned down b's buy-in")

	send(t, b, `{"action":"rebuy","amount":200}`)
	waitLog(t, aRecv, "b asks to rebuy for 200")
	send(t, a, `{"action":"approve-buy-in","name":"b"}`)
	waitLog(t, aRecv, "a approved b's rebuy")
	waitLog(t, aRecv, "b rebuys for 200")
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").BoughtIn == 300 })
	if r := row(l, "b"); r.BoughtIn != 300 || r.Stack != 300 {
		t.Errorf("b after the approved rebuy = %+v, want 300 in and on the table", r)
	}
}

func TestHostRebuysWithoutApproval(t *testing.T) {
	a, _, aRecv, _ := approvalPair(t, newTestServer(t))
	send(t, a, `{"action":"rebuy","amount":100}`)
	waitLog(t, aRecv, "a rebuys for 100")
}

func TestSittingBackDownNeedsApproval(t *testing.T) {
	srv := newTestServer(t)
	a, b, aRecv, bRecv := approvalPair(t, srv)
	waitToken(t, bRecv)
	send(t, b, `{"action":"leave-seat"}`)
	waitWatching(t, aRecv, "b")
	send(t, b, seatMsg(3))
	waitLog(t, aRecv, "b asks to sit back down with 100")
	send(t, a, `{"action":"approve-buy-in","name":"b"}`)
	waitLog(t, aRecv, "a approved b's buy-in")
	waitLog(t, aRecv, "b sits down with 100")
	// b's seat came back with a new token, after the empty one from leaving
	for token := waitToken(t, bRecv); token == ""; token = waitToken(t, bRecv) {
	}

	// a new name sits down without asking
	c, _ := dialRecv(t, srv)
	send(t, c, joinMsg("t", "c"), seatMsg(4))
	waitLog(t, aRecv, "c sits down with 100")
}

func TestApprovalDropsStaleRequests(t *testing.T) {
	srv := newTestServer(t)
	a, b, aRecv, _ := approvalPair(t, srv)
	send(t, b, `{"action":"leave-seat"}`, seatMsg(3))
	waitLog(t, aRecv, "b asks to sit back down with 100")
	c, _ := dialRecv(t, srv)
	send(t, c, joinMsg("t", "c"), seatMsg(3))
	waitLog(t, aRecv, "c sits down with 100")
	send(t, a, `{"action":"approve-buy-in","name":"b"}`)
	if got := waitError(t, aRecv); got != "seat 3 isn't free anymore, so their request was dropped" {
		t.Errorf("approving a taken seat: got %q", got)
	}
	send(t, a, `{"action":"approve-buy-in","name":"b"}`)
	if got := waitError(t, aRecv); got != "b hasn't asked to buy in" {
		t.Errorf("approving a dropped request: got %q", got)
	}
}

func TestLeavingDropsRequest(t *testing.T) {
	a, b, aRecv, _ := approvalPair(t, newTestServer(t))
	send(t, b, `{"action":"leave-seat"}`, seatMsg(3))
	waitLog(t, aRecv, "b asks to sit back down with 100")
	// drop older updates, which had no watchers before b sat down
	for len(aRecv.watch) > 0 {
		<-aRecv.watch
	}
	send(t, b, `{"action":"leave-table","tablename":"t"}`)
	waitWatching(t, aRecv)
	send(t, a, `{"action":"approve-buy-in","name":"b"}`)
	if got := waitError(t, aRecv); got != "b hasn't asked to buy in" {
		t.Errorf("approving after b left: got %q", got)
	}
}

func TestOnlyHostApproves(t *testing.T) {
	_, b, _, bRecv := approvalPair(t, newTestServer(t))
	for _, msg := range []string{`{"action":"approve-buy-in","name":"b"}`, `{"action":"deny-buy-in","name":"b"}`} {
		send(t, b, msg)
		if got := waitError(t, bRecv); got != "only the host can do that" {
			t.Errorf("%s: got %q", msg, got)
		}
	}
}
