package server

import (
	"testing"
	"time"
)

// session is the session state of a game update.
type session struct {
	paused, ending, ended bool
}

// waitSession waits for a game update with session state want.
func waitSession(t *testing.T, r received, want session) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case s := <-r.session:
			if s == want {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for session %+v", want)
		}
	}
}

// hands counts the hands dealt in logs.
func hands(logs []string) int {
	n := 0
	for _, l := range logs {
		if l == "starting new hand" {
			n++
		}
	}
	return n
}

func TestOnlyHostPausesAndEnds(t *testing.T) {
	_, b, _, bRecv := seatedPair(t)
	for _, msg := range []string{`{"action":"pause-game"}`, `{"action":"resume-game"}`, `{"action":"end-session"}`} {
		send(t, b, msg)
		if got := waitError(t, bRecv); got != "only the host can do that" {
			t.Errorf("%s: got %q", msg, got)
		}
	}
}

func TestPauseAfterHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)
	send(t, a, `{"action":"pause-game"}`)
	waitLog(t, aRecv, "a paused the game after this hand")
	waitSession(t, aRecv, session{paused: true})

	// the hand plays out, and no new one is dealt
	send(t, a, `{"action":"player-fold"}`)
	if n := hands(collectLogs(aRecv, 5*nextHandDelay)); n != 0 {
		t.Errorf("dealt %d hands while paused", n)
	}
	send(t, a, `{"action":"pause-game"}`)
	if got := waitError(t, aRecv); got != "the game is already paused" {
		t.Errorf("pausing twice: got %q", got)
	}

	send(t, a, `{"action":"resume-game"}`)
	waitLog(t, aRecv, "a resumed the game")
	waitLog(t, aRecv, "starting new hand")
}

func TestStartBlockedWhilePaused(t *testing.T) {
	a, _, aRecv, _ := seatedPair(t)
	send(t, a, `{"action":"pause-game"}`, `{"action":"start-game"}`)
	if got := waitError(t, aRecv); got != "the game is paused" {
		t.Errorf("start while paused: got %q", got)
	}
}

func TestEndSessionAfterHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, bRecv := seatTwo(t, srv)
	send(t, a, `{"action":"end-session"}`)
	waitLog(t, aRecv, "a will end the session after this hand")
	waitSession(t, aRecv, session{ending: true})

	send(t, a, `{"action":"player-fold"}`)
	waitLog(t, aRecv, "the session has ended, the ledger is final")
	waitSession(t, aRecv, session{ended: true})
	if n := hands(collectLogs(aRecv, 5*nextHandDelay)); n != 0 {
		t.Errorf("dealt %d hands after the session ended", n)
	}

	// nothing can deal, seat or move chips now, but the ledger is still there
	for _, msg := range []string{`{"action":"rebuy","amount":100}`, `{"action":"start-game"}`, `{"action":"sit-out"}`} {
		send(t, b, msg)
		if got := waitError(t, bRecv); got != "the session has ended" {
			t.Errorf("%s after the end: got %q", msg, got)
		}
	}
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "a").Net != 0 })
	if row(l, "a").Net != -10 || row(l, "b").Net != 10 {
		t.Errorf("final ledger = %+v", l.Rows)
	}
}

func TestEndedTableKeptThenClosed(t *testing.T) {
	srv, hub := newTestServerHub(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1), `{"action":"end-session"}`)
	waitLog(t, aRecv, "a ended the session")

	// everyone leaves, but the table stays readable
	send(t, a, `{"action":"leave-table","tablename":"t"}`)
	time.Sleep(endedTableTTL / 5)
	waitForTables(t, hub, "t")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"))
	l := waitLedger(t, bRecv, func(l ledgerView) bool { return len(l.Rows) == 1 })
	if row(l, "a").CashedOut != 100 {
		t.Errorf("ledger after the end = %+v", l.Rows)
	}

	// then it closes, telling whoever is still there
	waitError(t, bRecv)
	if code := <-bRecv.codes; code != codeClosed {
		t.Errorf("b got code %q, want %q", code, codeClosed)
	}
	waitForTables(t, hub)
}

func TestEndedSessionClosesBots(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, addBotMsg(2, "harper"), `{"action":"end-session"}`)
	waitLog(t, aRecv, "a ended the session")
	for _, msg := range []string{addBotMsg(3, "eric"), `{"action":"remove-bot","seatID":2}`} {
		send(t, a, msg)
		if got := waitError(t, aRecv); got != "the session has ended" {
			t.Errorf("%s after the end: got %q", msg, got)
		}
	}
}
