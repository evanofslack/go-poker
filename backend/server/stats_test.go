package server

import (
	"slices"
	"strings"
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
	if h.Hand != 1 || h.Started == 0 || h.Stacks["a"] != 90 || h.Stacks["b"] != 110 {
		t.Errorf("hand = %+v, want hand 1, started, with a on 90 and b on 110", h)
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
		recordHand(tb, tb.game.GenerateOmniView(), nil)
	}
	if len(tb.hands) != maxHands || tb.hands[0].Hand != 6 || tb.hands[maxHands-1].Hand != maxHands+5 {
		t.Errorf("kept %d hands, %d to %d; want the last %d", len(tb.hands), tb.hands[0].Hand, tb.hands[len(tb.hands)-1].Hand, maxHands)
	}
}

// played builds a hand from actions like "preflop a raise", after the blinds:
// a posts the small blind and b the big blind.
func played(board int, actions ...string) *handRecord {
	r := &handRecord{
		Players: []handPlayer{{Name: "a"}, {Name: "b"}},
		Actions: []handAction{
			{Street: "preflop", Name: "a", Kind: "small blind"},
			{Street: "preflop", Name: "b", Kind: "big blind"},
		},
		Board: make([]string, board),
	}
	for _, a := range actions {
		f := strings.Fields(a)
		r.Actions = append(r.Actions, handAction{Street: f[0], Name: f[1], Kind: f[2]})
	}
	return r
}

func TestTally(t *testing.T) {
	tests := []struct {
		name string
		hand *handRecord
		a, b playerTotals
	}{
		{
			name: "limp and check down",
			hand: played(5, "preflop a call", "preflop b check", "flop b check", "flop a bet", "flop b call"),
			a:    playerTotals{Hands: 1, VPIP: 1, Aggressive: 1, Calls: 1, SawFlop: 1},
			b:    playerTotals{Hands: 1, Calls: 1, SawFlop: 1},
		},
		{
			name: "raise and fold",
			hand: played(0, "preflop a raise", "preflop b fold"),
			a:    playerTotals{Hands: 1, VPIP: 1, PFR: 1, Aggressive: 1},
			b:    playerTotals{Hands: 1, ThreeBetSpots: 1},
		},
		{
			name: "3-bet and call",
			hand: played(3, "preflop a raise", "preflop b raise", "preflop a call", "flop b check", "flop a check"),
			a:    playerTotals{Hands: 1, VPIP: 1, PFR: 1, Aggressive: 1, Calls: 1, SawFlop: 1},
			b:    playerTotals{Hands: 1, VPIP: 1, PFR: 1, ThreeBetSpots: 1, ThreeBets: 1, Aggressive: 1, SawFlop: 1},
		},
		{
			name: "walk",
			hand: played(0, "preflop a fold"),
			a:    playerTotals{Hands: 1},
			b:    playerTotals{Hands: 1},
		},
	}
	for _, tt := range tests {
		totals := map[string]*playerTotals{}
		tally(totals, tt.hand)
		if *totals["a"] != tt.a {
			t.Errorf("%s: a = %+v, want %+v", tt.name, *totals["a"], tt.a)
		}
		if *totals["b"] != tt.b {
			t.Errorf("%s: b = %+v, want %+v", tt.name, *totals["b"], tt.b)
		}
	}
}

func TestTallyShowdowns(t *testing.T) {
	totals := map[string]*playerTotals{}
	show := func(winner string, aHand string, aScore int, bHand string, bScore int) {
		r := played(5, "preflop a call", "preflop b check")
		r.Players[0].Shown, r.Players[0].Hand, r.Players[0].score = true, aHand, aScore
		r.Players[1].Shown, r.Players[1].Hand, r.Players[1].score = true, bHand, bScore
		r.Pots = []handPot{{Amount: 40, Winners: []string{winner}}}
		if winner == "a" {
			r.Players[0].Net, r.Players[1].Net = 20, -20
		} else {
			r.Players[0].Net, r.Players[1].Net = -20, 20
		}
		tally(totals, r)
	}
	show("a", "a flush, ace high", 400, "a pair of twos", 6000)
	show("b", "ace high", 6200, "a full house, jacks full of fours", 200)
	show("a", "a straight, ten high", 1605, "high card", 7000)

	a, b := totals["a"], totals["b"]
	if a.Showdowns != 3 || a.ShowdownWins != 2 || b.ShowdownWins != 1 {
		t.Errorf("showdowns: a %d won %d, b won %d; want 3, 2 and 1", a.Showdowns, a.ShowdownWins, b.ShowdownWins)
	}
	if a.BestHand != "a flush, ace high" || b.BestHand != "a full house, jacks full of fours" {
		t.Errorf("best hands = %q and %q", a.BestHand, b.BestHand)
	}
	if a.Net != 20 || b.Net != -20 {
		t.Errorf("nets = %d and %d, want 20 and -20", a.Net, b.Net)
	}
}

func TestTotalsSent(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)
	send(t, a, `{"action":"player-fold"}`)
	s := waitStats(t, aRecv, func(s updateStats) bool { return !s.Full })
	if got := s.Totals["a"]; got.Hands != 1 || got.Net != -10 {
		t.Errorf("a's totals after folding = %+v, want 1 hand and -10", got)
	}
	if got := s.Totals["b"]; got.Hands != 1 || got.Net != 10 || got.VPIP != 0 {
		t.Errorf("b's totals after a walk = %+v, want 1 hand, +10 and no VPIP", got)
	}
}
