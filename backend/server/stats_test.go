package server

import (
	"slices"
	"testing"
	"time"
)

// waitStats waits for a stats message that passes ok.
func waitStats(t *testing.T, r received, ok func(updateStats) bool) updateStats {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case s := <-r.stats:
			if ok(s) {
				return s
			}
		case <-timeout:
			t.Fatal("timed out waiting for stats")
		}
	}
}

func TestHandSnapshots(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)
	if s := waitStats(t, aRecv, func(s updateStats) bool { return s.Full }); len(s.Hands) != 0 {
		t.Errorf("stats on joining a new table = %+v, want none", s.Hands)
	}

	send(t, a, `{"action":"player-fold"}`)
	s := waitStats(t, aRecv, func(s updateStats) bool { return !s.Full })
	if len(s.Hands) != 1 {
		t.Fatalf("got %d hands at the end of a hand, want just the new one", len(s.Hands))
	}
	h := s.Hands[0]
	if h.Hand != 1 || h.Stacks["a"] != 90 || h.Stacks["b"] != 110 {
		t.Errorf("hand = %+v, want hand 1 with a on 90 and b on 110", h)
	}
	if !slices.Equal(slices.Sorted(slices.Values(h.Played)), []string{"a", "b"}) {
		t.Errorf("played = %v, want a and b", h.Played)
	}
	if h.Won["b"] != 30 || h.Won["a"] != 0 {
		t.Errorf("won = %v, want b taking the whole 30 pot", h.Won)
	}

	// a late joiner gets every hand so far
	c, cRecv := dialRecv(t, srv)
	send(t, c, joinMsg("t", "c"))
	s = waitStats(t, cRecv, func(s updateStats) bool { return s.Full })
	if len(s.Hands) != 1 || s.Hands[0].Hand != 1 {
		t.Errorf("late joiner got %+v, want hand 1", s.Hands)
	}
}

func TestHandSnapshotsCapped(t *testing.T) {
	tb := newTable("t", defaultSettings)
	go tb.run() // takes the broadcasts
	defer close(tb.done)
	for range maxHands + 5 {
		recordHand(tb, tb.game.GenerateOmniView())
	}
	if len(tb.hands) != maxHands || tb.hands[0].Hand != 6 || tb.hands[maxHands-1].Hand != maxHands+5 {
		t.Errorf("kept %d hands, %d to %d; want the last %d", len(tb.hands), tb.hands[0].Hand, tb.hands[len(tb.hands)-1].Hand, maxHands)
	}
}
