package server

import (
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/poker"
)

// waitHistory waits for a hand history message that passes ok.
func waitHistory(t *testing.T, r received, ok func(handHistory) bool) handHistory {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case h := <-r.history:
			if ok(h) {
				return h
			}
		case <-timeout:
			t.Fatal("timed out waiting for hand history")
		}
	}
}

// newHand waits for the record of a hand that just ended.
func newHand(t *testing.T, r received) handRecord {
	t.Helper()
	h := waitHistory(t, r, func(h handHistory) bool { return !h.Full && len(h.Hands) == 1 })
	return h.Hands[0]
}

func player(r handRecord, name string) handPlayer {
	for _, p := range r.Players {
		if p.Name == name {
			return p
		}
	}
	return handPlayer{}
}

func TestHistoryHidesFoldedCards(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, bRecv := seatTwo(t, srv)
	s, sRecv := dialRecv(t, srv)
	send(t, s, joinMsg("t", "s"))
	waitGame(t, sRecv.games, func(poker.GameView) bool { return true })

	send(t, a, `{"action":"player-fold"}`)
	forA, forB, forS := newHand(t, aRecv), newHand(t, bRecv), newHand(t, sRecv)

	if c := player(forA, "a").Cards; len(c) != 2 {
		t.Errorf("a's own cards = %v, want 2", c)
	}
	if c := player(forB, "b").Cards; len(c) != 2 {
		t.Errorf("b's own cards = %v, want 2", c)
	}
	if c := player(forB, "a").Cards; c != nil {
		t.Errorf("b sees a's folded cards %v", c)
	}
	if c := player(forA, "b").Cards; c != nil {
		t.Errorf("a sees b's unshown cards %v", c)
	}
	for _, p := range forS.Players {
		if p.Cards != nil || p.Shown {
			t.Errorf("spectator sees %s's cards %v", p.Name, p.Cards)
		}
	}

	r := forS
	kinds := []string{}
	for _, a := range r.Actions {
		kinds = append(kinds, a.Street+" "+a.Name+" "+a.Kind)
	}
	if want := []string{"preflop a small blind", "preflop b big blind", "preflop a fold"}; !slices.Equal(kinds, want) {
		t.Errorf("actions = %q, want %q", kinds, want)
	}
	if r.Hand != 1 || r.Dealer != "a" || len(r.Board) != 0 {
		t.Errorf("hand %d, dealer %q, board %v; want hand 1, dealer a, no board", r.Hand, r.Dealer, r.Board)
	}
	if a, b := player(r, "a"), player(r, "b"); a.Stack != 100 || a.Net != -10 || b.Net != 10 {
		t.Errorf("a started on %d and netted %d, b netted %d; want 100, -10 and 10", a.Stack, a.Net, b.Net)
	}
	if len(r.Pots) != 1 || r.Pots[0].Amount != 30 || !slices.Equal(r.Pots[0].Winners, []string{"b"}) || r.Pots[0].Hand != "" {
		t.Errorf("pots = %+v, want 30 to b without a showdown", r.Pots)
	}
}

func TestHistoryShowsShowdown(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, bRecv := seatTwo(t, srv)
	s, sRecv := dialRecv(t, srv)
	send(t, s, joinMsg("t", "s"))
	waitGame(t, sRecv.games, func(poker.GameView) bool { return true })

	// a calls and b checks, then both check down; b acts first after the flop
	turn := func(stage poker.GameStage, name string) {
		t.Helper()
		waitGame(t, aRecv.games, func(g poker.GameView) bool {
			return g.Betting && g.Stage == stage && g.Players[g.ActionNum].Username == name
		})
	}
	send(t, a, `{"action":"player-call"}`)
	turn(poker.PreFlop, "b")
	send(t, b, `{"action":"player-check"}`)
	for _, stage := range []poker.GameStage{poker.Flop, poker.Turn, poker.River} {
		turn(stage, "b")
		send(t, b, `{"action":"player-check"}`)
		turn(stage, "a")
		send(t, a, `{"action":"player-check"}`)
	}

	for name, recv := range map[string]received{"a": aRecv, "b": bRecv, "s": sRecv} {
		r := newHand(t, recv)
		for _, p := range r.Players {
			if len(p.Cards) != 2 || !p.Shown || p.Hand == "" {
				t.Errorf("%s sees %s with cards %v, shown %v, hand %q; want both shown and named", name, p.Name, p.Cards, p.Shown, p.Hand)
			}
		}
		if len(r.Board) != 5 {
			t.Errorf("%s sees board %v, want 5 cards", name, r.Board)
		}
		if n := player(r, "a").Net + player(r, "b").Net; n != 0 {
			t.Errorf("nets add up to %d, want 0", n)
		}
		var streets []string
		for _, a := range r.Actions {
			if len(streets) == 0 || streets[len(streets)-1] != a.Street {
				streets = append(streets, a.Street)
			}
		}
		if want := []string{"preflop", "flop", "turn", "river"}; !slices.Equal(streets, want) {
			t.Errorf("streets = %q, want %q", streets, want)
		}
		if len(r.Pots) == 0 || r.Pots[0].Hand == "" {
			t.Errorf("pots = %+v, want a named winning hand", r.Pots)
		}
	}
}

func TestHistoryOnJoin(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, bRecv := seatTwo(t, srv)
	token := waitToken(t, bRecv)
	if h := waitHistory(t, aRecv, func(h handHistory) bool { return h.Full }); len(h.Hands) != 0 || h.More {
		t.Errorf("history on joining a new table = %+v, want none", h)
	}
	send(t, a, `{"action":"player-fold"}`)
	newHand(t, aRecv)

	// b comes back on a new connection and still sees its own cards
	b.Close()
	b2, b2Recv := dialRecv(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"b","token":"`+token+`"}`)
	h := waitHistory(t, b2Recv, func(h handHistory) bool { return h.Full })
	if len(h.Hands) != 1 || len(player(h.Hands[0], "b").Cards) != 2 || player(h.Hands[0], "a").Cards != nil {
		t.Errorf("reconnected b's history = %+v, want hand 1 with only b's cards", h.Hands)
	}

	// someone new sees no cards
	c, cRecv := dialRecv(t, srv)
	send(t, c, joinMsg("t", "c"))
	h = waitHistory(t, cRecv, func(h handHistory) bool { return h.Full })
	if len(h.Hands) != 1 {
		t.Fatalf("new client got %d hands, want 1", len(h.Hands))
	}
	for _, p := range h.Hands[0].Players {
		if p.Cards != nil {
			t.Errorf("new client sees %s's cards %v", p.Name, p.Cards)
		}
	}
}

func TestHistoryFor(t *testing.T) {
	tbl := newTable("t", defaultSettings)
	for i := 1; i <= 45; i++ {
		tbl.history = append(tbl.history, handRecord{Hand: i})
	}
	tests := []struct {
		before      int
		first, last int
		more        bool
	}{
		{0, 26, 45, true},
		{26, 6, 25, true},
		{6, 1, 5, false},
		{1, 0, 0, false},
		{1000, 26, 45, true},
	}
	for _, tt := range tests {
		hands, more := historyFor(tbl, "", tt.before)
		first, last := 0, 0
		if len(hands) > 0 {
			first, last = hands[0].Hand, hands[len(hands)-1].Hand
		}
		if first != tt.first || last != tt.last || more != tt.more {
			t.Errorf("historyFor(before %d) = hands %d to %d, more %v; want %d to %d, more %v",
				tt.before, first, last, more, tt.first, tt.last, tt.more)
		}
	}
}

func TestViewFor(t *testing.T) {
	r := handRecord{Players: []handPlayer{
		{Name: "a", UUID: "ua", Cards: []string{"As", "Ks"}, Hand: "ace high"},
		{Name: "b", UUID: "ub", Cards: []string{"2c", "2d"}, Shown: true, Hand: "a pair of twos"},
		{Name: "c", UUID: "uc", Cards: []string{"7h", "8h"}},
	}}
	got := r.viewFor("ua")
	if got.Players[0].Cards == nil || got.Players[1].Cards == nil || got.Players[2].Cards != nil {
		t.Errorf("a's view = %+v, want a's and b's cards only", got.Players)
	}
	got = r.viewFor("")
	if got.Players[0].Cards != nil || got.Players[0].Hand != "" || got.Players[1].Cards == nil {
		t.Errorf("spectator view = %+v, want only b's shown cards", got.Players)
	}
	if r.Players[0].Cards == nil {
		t.Error("viewFor changed the record it was called on")
	}
}

func TestAggressor(t *testing.T) {
	r := &handRecord{Actions: []handAction{
		{Street: "preflop", Name: "a", Kind: "small blind"},
		{Street: "preflop", Name: "b", Kind: "big blind"},
		{Street: "preflop", Name: "c", Kind: "raise"},
		{Street: "preflop", Name: "a", Kind: "call"},
		{Street: "flop", Name: "a", Kind: "check"},
		{Street: "flop", Name: "c", Kind: "check"},
		{Street: "turn", Name: "a", Kind: "bet"},
	}}
	for street, want := range map[string]string{"preflop": "", "flop": "c", "turn": "c", "river": "a"} {
		if got := r.aggressor(street); got != want {
			t.Errorf("aggressor before %s = %q, want %q", street, got, want)
		}
	}
	if got := (*handRecord)(nil).aggressor("flop"); got != "" {
		t.Errorf("aggressor with no hand = %q", got)
	}
}
