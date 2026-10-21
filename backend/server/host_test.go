package server

import (
	"testing"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/gorilla/websocket"
)

// waitHost waits for a game update whose host is name.
func waitHost(t *testing.T, r received, name string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case h := <-r.hosts:
			if h == name {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for host %q", name)
		}
	}
}

// seatedPair creates a table with host a in seat 1 and b in seat 2, without
// starting.
func seatedPair(t *testing.T) (a, b *websocket.Conn, aRecv, bRecv received) {
	t.Helper()
	srv := newTestServer(t)
	a, aRecv = createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	b, bRecv = dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	return a, b, aRecv, bRecv
}

func TestHostPassesOnLeave(t *testing.T) {
	a, _, _, bRecv := seatedPair(t)
	send(t, a, `{"action":"leave-table","tablename":"t"}`)
	waitHost(t, bRecv, "b")
}

func TestHostKeptDuringGrace(t *testing.T) {
	a, _, _, bRecv := seatedPair(t)
	a.Close()
	waitLog(t, bRecv, "a disconnected")
	time.Sleep(reconnectGrace / 3)
	for len(bRecv.hosts) > 0 {
		if h := <-bRecv.hosts; h != "a" {
			t.Fatalf("host moved to %q during a's grace period", h)
		}
	}
	waitHost(t, bRecv, "b")
}

func TestHostKeptOnReconnect(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	token := waitToken(t, aRecv)
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	a.Close()
	waitLog(t, bRecv, "a disconnected")
	a2, a2Recv := dialRecv(t, srv)
	send(t, a2, `{"action":"join-table","tablename":"t","username":"a","token":"`+token+`"}`)
	waitHost(t, a2Recv, "a")
	time.Sleep(2 * reconnectGrace)
	send(t, a2, `{"action":"set-banker","name":"b"}`)
	waitLog(t, bRecv, "a made b the banker")
}

func TestSpectatorHostPassesOnDisconnect(t *testing.T) {
	srv := newTestServer(t)
	a, _ := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, bRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	a.Close()
	waitHost(t, bRecv, "b")
}

func TestOnlyHostActs(t *testing.T) {
	_, b, _, bRecv := seatedPair(t)
	for _, msg := range []string{
		`{"action":"make-host","name":"b"}`,
		`{"action":"set-banker","name":"b"}`,
		`{"action":"move-to-spectator","name":"a"}`,
		`{"action":"remove-player","name":"a"}`,
		`{"action":"sit-out-player","name":"a"}`,
		`{"action":"adjust-chips","name":"b","amount":100}`,
	} {
		send(t, b, msg)
		if got := waitError(t, bRecv); got != "only the host can do that" {
			t.Errorf("%s: got %q", msg, got)
		}
	}
}

func TestMakeHostAndBanker(t *testing.T) {
	a, b, aRecv, bRecv := seatedPair(t)
	send(t, a, `{"action":"set-banker","name":"zed"}`)
	if got := waitError(t, aRecv); got != "zed isn't at the table" {
		t.Errorf("unknown banker: got %q", got)
	}
	send(t, a, `{"action":"make-host","name":"b"}`)
	waitHost(t, aRecv, "b")
	// the banker follows the host until someone picks one
	waitLedger(t, aRecv, func(l ledgerView) bool { return l.Banker == "b" })

	send(t, a, `{"action":"set-banker","name":"a"}`)
	if got := waitError(t, aRecv); got != "only the host can do that" {
		t.Errorf("old host: got %q", got)
	}
	send(t, b, `{"action":"set-banker","name":"a"}`)
	waitLedger(t, bRecv, func(l ledgerView) bool { return l.Banker == "a" })
	send(t, b, `{"action":"make-host","name":"a"}`)
	waitHost(t, bRecv, "a")
	if l := waitLedger(t, bRecv, func(ledgerView) bool { return true }); l.Banker != "a" {
		t.Errorf("banker = %q after the host changed, want the picked a", l.Banker)
	}
}

func TestMoveToSpectator(t *testing.T) {
	a, _, aRecv, bRecv := seatedPair(t)
	waitToken(t, bRecv)
	send(t, a, `{"action":"move-to-spectator","name":"b"}`)
	if token := waitToken(t, bRecv); token != "" {
		t.Errorf("b got token %q, want none once watching", token)
	}
	waitWatching(t, aRecv, "b")
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").CashedOut == 100 })
	if l.Balance != 0 {
		t.Errorf("balance = %d, want 0", l.Balance)
	}
	waitLog(t, aRecv, "a moved b to watching")
}

func TestRemovePlayer(t *testing.T) {
	a, _, aRecv, bRecv := seatedPair(t)
	send(t, a, `{"action":"remove-player","name":"a"}`)
	if got := waitError(t, aRecv); got != "you can't remove yourself" {
		t.Errorf("removing yourself: got %q", got)
	}
	send(t, a, `{"action":"remove-player","name":"b"}`)
	waitError(t, bRecv)
	if code := <-bRecv.codes; code != codeRemoved {
		t.Errorf("b got code %q, want %q", code, codeRemoved)
	}
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	waitLog(t, aRecv, "a removed b from the table")
}

func TestSitOutPlayer(t *testing.T) {
	a, _, aRecv, _ := seatedPair(t)
	send(t, a, `{"action":"sit-out-player","name":"b"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool {
		_, ready := byName(t, g, "b")
		return !ready
	})
	waitLog(t, aRecv, "a sat b out")
	send(t, a, `{"action":"sit-out-player","name":"b"}`)
	if got := waitError(t, aRecv); got != "b is already sitting out" {
		t.Errorf("sitting out twice: got %q", got)
	}
}

func TestAdjustChips(t *testing.T) {
	a, _, aRecv, _ := seatedPair(t)
	send(t, a, `{"action":"adjust-chips","name":"b","amount":50,"reason":"miscount"}`)
	waitLog(t, aRecv, "a added 50 to b's stack (miscount)")
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 150 })
	if r := row(l, "b"); r.BoughtIn != 150 || r.Net != 0 {
		t.Errorf("b after the add = %+v, want bought in 150 and net 0", r)
	}

	send(t, a, `{"action":"adjust-chips","name":"b","amount":-200}`)
	if got := waitError(t, aRecv); got != "b only has 150" {
		t.Errorf("taking too much: got %q", got)
	}
	send(t, a, `{"action":"adjust-chips","name":"b","amount":-150}`)
	waitLog(t, aRecv, "a took 150 from b's stack")
	l = waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 0 })
	if r := row(l, "b"); r.CashedOut != 150 || l.Balance != 0 {
		t.Errorf("b after the take = %+v, balance %d", r, l.Balance)
	}
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool {
		for _, p := range g.Players {
			if p.Username == "b" {
				return p.Stack == 0
			}
		}
		return false
	})
	if _, ready := byName(t, g, "b"); ready {
		t.Error("b is still dealt in with no chips")
	}

	send(t, a, `{"action":"adjust-chips","name":"b","amount":0}`)
	if got := waitError(t, aRecv); got != "invalid chip change" {
		t.Errorf("zero change: got %q", got)
	}
}

func TestAdjustChipsMidHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)
	send(t, a, `{"action":"adjust-chips","name":"b","amount":50}`)
	if got := waitError(t, aRecv); got != "wait for the hand to end" {
		t.Errorf("mid-hand change: got %q", got)
	}
}
