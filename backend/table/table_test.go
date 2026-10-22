package table

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// pair is a table with a and b seated in seats 1 and 2, not started.
func pair(t *testing.T) *harness {
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	return h
}

func TestJoin(t *testing.T) {
	h := newHarness(t, DefaultSettings, "secret")
	if res := h.do(Join{ID: "x", Name: "a"}); res.Joined != nil {
		t.Fatal("joined without the password")
	}
	if got := sentTo[Error](h, "x"); got[0].Code != "password-required" {
		t.Fatalf("got %+v", got)
	}
	h.do(Join{ID: "y", Name: "a", Password: "nope"})
	if got := sentTo[Error](h, "y"); got[0].Code != "wrong-password" {
		t.Fatalf("got %+v", got)
	}
	if res := h.do(Join{ID: "z", Name: "a", Password: "secret"}); res.Joined == nil {
		t.Fatal("couldn't join with the password")
	}
	if h.s.Host != "a" {
		t.Fatal("the first person in should host")
	}
	if res := h.do(Join{ID: "w", Name: "A", Password: "secret"}); res.Joined != nil {
		t.Fatal("joined with a taken name in another case")
	}
	if got := sentTo[ChatMessage](h, ""); got[0].Message != "a has joined" {
		t.Fatalf("chat %+v", got)
	}
}

func TestHandFlow(t *testing.T) {
	h := pair(t)
	h.refused(Start{Person: "nobody"})
	h.ok(Start{Person: h.ids["a"]})
	if !h.s.Game.InHand() || h.s.Game.Hands != 1 {
		t.Fatal("start should deal")
	}
	h.play(holdem.Call{})
	h.play(holdem.Check{})
	h.checkInvariants()
	if h.s.Game.Street != holdem.Preflop {
		t.Fatal("the flop should wait for the street delay")
	}
	h.advance(time.Second)
	if h.s.Game.Street != holdem.Flop || h.s.Acts[1] != (Act{}) && h.s.Acts[1].Street != holdem.Preflop {
		t.Fatalf("flop not dealt: %s", h)
	}
	h.refused(Play{Person: h.ids["a"], Action: holdem.Check{}}) // b acts first after the flop
	h.play(holdem.Raise{To: 100})
	h.play(holdem.Fold{})
	if h.s.Game.InHand() {
		t.Fatal("hand should be over")
	}
	if h.stack("b") != 1020 || h.stack("a") != 980 {
		t.Fatalf("stacks a %d b %d", h.stack("a"), h.stack("b"))
	}
	// the next hand comes after the delay
	h.advance(4 * time.Second)
	if h.s.Game.InHand() {
		t.Fatal("dealt too soon")
	}
	h.advance(time.Second)
	if h.s.Game.Hands != 2 {
		t.Fatal("next hand not dealt")
	}
	h.checkInvariants()

	if len(h.s.History) != 1 {
		t.Fatalf("history %d", len(h.s.History))
	}
	rec := h.s.History[0]
	want := []string{"small blind", "big blind", "call", "check", "bet", "fold"}
	var kinds []string
	for _, a := range rec.Actions {
		kinds = append(kinds, a.Kind)
	}
	if !reflect.DeepEqual(kinds, want) || rec.Pots[0].Winners[0] != "b" || rec.Players[1].Net != 20 {
		t.Fatalf("record %+v", rec)
	}
	if h.s.Totals["a"].Hands != 1 || h.s.Stats[0].Won["b"] != 140 {
		t.Fatalf("totals %+v stats %+v", h.s.Totals["a"], h.s.Stats[0])
	}
}

func TestTurnTimeout(t *testing.T) {
	h := pair(t)
	h.ok(Start{Person: h.ids["a"]})
	// a, on the button, owes the small blind's difference: folds
	h.advance(30 * time.Second)
	if h.s.Game.Hands != 1 || h.s.Game.InHand() {
		t.Fatalf("a should have been folded: %s", h)
	}
	last := h.s.History[0].Actions[2]
	if last.Kind != "fold" || !last.Timeout {
		t.Fatalf("last action %+v", last)
	}
	// the turn timer checks when it can
	h.advance(5 * time.Second)
	h.play(holdem.Call{})
	h.advance(30 * time.Second)
	if a := h.s.Current.Actions[3]; a.Kind != "check" || !a.Timeout {
		t.Fatalf("action %+v", a)
	}
}

func TestLeaveSeatMidHand(t *testing.T) {
	h := pair(t)
	h.join("c")
	h.sit("c", 3, 1000)
	h.ok(Start{Person: h.ids["a"]})
	// a is first to act preflop with three players; c leaves before then
	h.ok(LeaveSeat{Person: h.ids["c"]})
	if h.s.People[h.ids["c"]].Seat != 0 || !h.s.Seats[3].Leaving {
		t.Fatal("c should be watching, their seat leaving")
	}
	h.refused(LeaveSeat{Person: h.ids["c"]})
	h.play(holdem.Call{})
	h.play(holdem.Call{}) // b, the small blind; c, the big blind, is folded
	if h.s.Game.InHand() {
		h.advance(time.Second)
	}
	for h.s.Game.InHand() {
		if h.s.Game.Betting {
			h.play(holdem.Check{})
		} else {
			h.advance(time.Second)
		}
	}
	if h.s.Seats[3] != nil {
		t.Fatal("c's seat should be gone after the hand")
	}
	h.mustLog("c left the table")
	if row := Ledger(h.s).Rows[2]; row.Name != "c" || row.CashedOut != 980 {
		t.Fatalf("ledger %+v", row)
	}
	h.checkInvariants()
}

func TestReconnect(t *testing.T) {
	h := pair(t)
	h.ok(Disconnect{Person: h.ids["a"]})
	h.mustLog("a disconnected")
	if v := ViewFor(h.s, h.ids["b"], h.now); !v.Players[0].Disconnected || h.s.Host != "a" {
		t.Fatal("a keeps their seat and host while disconnected")
	}
	h.checkInvariants()
	res := h.do(Join{ID: "new", Name: "whatever", Token: h.tokens["a"]})
	if res.Joined == nil || res.Joined.ID != h.ids["a"] || !res.Joined.Reclaimed || res.Grace != "reconnected" {
		t.Fatalf("reclaim: %+v", res.Joined)
	}
	h.checkInvariants()

	h.ok(Disconnect{Person: h.ids["a"]})
	h.advance(time.Minute)
	if h.s.People[h.ids["a"]] != nil || h.s.Seats[1] != nil || h.s.Host != "b" {
		t.Fatalf("grace expired: host %q", h.s.Host)
	}
	if h.do(Join{ID: "late", Name: "a2", Token: h.tokens["a"]}).Joined.Reclaimed {
		t.Fatal("an expired token reclaimed a seat")
	}
	h.checkInvariants()
}

func TestLastOneOutCloses(t *testing.T) {
	h := pair(t)
	h.ok(Leave{Person: h.ids["a"]})
	if h.s.Host != "b" || h.closed != nil {
		t.Fatal("host passes to b")
	}
	h.ok(Disconnect{Person: h.ids["b"]})
	if h.closed != nil {
		t.Fatal("b's seat waits for a reconnect")
	}
	h.advance(time.Minute)
	if h.closed == nil || h.closed.Reason != "empty" {
		t.Fatalf("closed %+v", h.closed)
	}
}

func TestSitOut(t *testing.T) {
	h := pair(t)
	h.join("c")
	h.sit("c", 3, 1000)
	h.ok(Start{Person: h.ids["a"]})
	h.ok(SitOut{Person: h.ids["b"]})
	h.mustLog("b will sit out the next hand")
	h.ok(SitOut{Person: h.ids["b"]})
	h.mustLog("b will play the next hand")
	h.ok(SitOut{Person: h.ids["b"]})
	h.play(holdem.Fold{})
	h.play(holdem.Fold{})
	if !h.s.Game.Player(2).SittingOut {
		t.Fatal("b should sit out once the hand ends")
	}
	h.advance(5 * time.Second)
	if h.s.Game.Player(2).In {
		t.Fatal("b was dealt in")
	}
	h.ok(SitOut{Person: h.ids["b"]})
	h.mustLog("b is back")
}

func TestRebuy(t *testing.T) {
	h := pair(t)
	h.refused(Rebuy{Person: h.ids["a"], Amount: 4000})
	h.ok(Rebuy{Person: h.ids["a"], Amount: 3000})
	if h.stack("a") != 4000 {
		t.Fatal("rebuy didn't add chips")
	}
	h.ok(Start{Person: h.ids["a"]})
	if msg := h.refused(Rebuy{Person: h.ids["b"], Amount: 10}); msg != "can't rebuy during a hand" {
		t.Fatal(msg)
	}
}

func TestApproveRebuys(t *testing.T) {
	set := DefaultSettings
	set.ApproveRebuys = true
	h := newHarness(t, set, "")
	h.join("host")
	h.join("b")
	h.sit("b", 2, 1000) // first buy-in needs no approval
	h.ok(Rebuy{Person: h.ids["b"], Amount: 500})
	if h.stack("b") != 1000 || len(h.s.Requests) != 1 {
		t.Fatal("rebuy should wait for the host")
	}
	h.refused(HostAction{Person: h.ids["b"], Kind: ApproveBuy, Name: "b"})
	h.ok(HostAction{Person: h.ids["host"], Kind: ApproveBuy, Name: "b"})
	if h.stack("b") != 1500 {
		t.Fatal("approved rebuy not applied")
	}

	// leaving and sitting back down asks too
	h.ok(LeaveSeat{Person: h.ids["b"]})
	h.ok(TakeSeat{Person: h.ids["b"], Seat: 3, BuyIn: 1000})
	if h.s.People[h.ids["b"]].Seat != 0 || h.s.Requests["b"].Seat != 3 {
		t.Fatal("sitting back down should wait for the host")
	}
	h.ok(HostAction{Person: h.ids["host"], Kind: DenyBuy, Name: "b"})
	h.ok(TakeSeat{Person: h.ids["b"], Seat: 3, BuyIn: 1000})
	h.join("c")
	h.sit("c", 3, 1000)
	h.do(HostAction{Person: h.ids["host"], Kind: ApproveBuy, Name: "b"})
	if len(h.s.Requests) != 0 || h.s.People[h.ids["b"]].Seat != 0 {
		t.Fatal("a request for a taken seat is dropped")
	}
	h.checkInvariants()
}

func TestHostControls(t *testing.T) {
	h := pair(t)
	h.join("c")
	if msg := h.refused(HostAction{Person: h.ids["b"], Kind: MakeHost, Name: "b"}); msg != "only the host can do that" {
		t.Fatal(msg)
	}
	h.ok(HostAction{Person: h.ids["a"], Kind: SetBanker, Name: "b"})
	if Ledger(h.s).Banker != "b" {
		t.Fatal("banker not set")
	}
	h.ok(AdjustChips{Person: h.ids["a"], Name: "b", Amount: 500, Reason: "late"})
	h.ok(AdjustChips{Person: h.ids["a"], Name: "b", Amount: -1500})
	if !h.s.Game.Player(2).SittingOut {
		t.Fatal("taking every chip sits them out")
	}
	h.mustLog("a added 500 to b's stack (late)")
	h.ok(HostAction{Person: h.ids["a"], Kind: MoveToSpectator, Name: "b"})
	if h.s.Seats[2] != nil || h.s.People[h.ids["b"]].Seat != 0 {
		t.Fatal("b should be watching")
	}
	h.ok(HostAction{Person: h.ids["a"], Kind: RemovePlayer, Name: "c"})
	if got := sentTo[Error](h, h.ids["c"]); got[len(got)-1].Code != CodeRemoved {
		t.Fatalf("c not told: %+v", got)
	}
	h.ok(HostAction{Person: h.ids["a"], Kind: MakeHost, Name: "b"})
	if h.s.Host != "b" {
		t.Fatal("host not passed")
	}
	h.checkInvariants()
}

func TestPauseAndEnd(t *testing.T) {
	h := pair(t)
	h.ok(Start{Person: h.ids["a"]})
	h.ok(Pause{Person: h.ids["a"], Pause: true})
	h.mustLog("paused the game after this hand")
	h.play(holdem.Fold{})
	h.advance(time.Minute)
	if h.s.Game.Hands != 1 {
		t.Fatal("dealt while paused")
	}
	h.ok(Pause{Person: h.ids["a"], Pause: false})
	h.advance(5 * time.Second)
	if h.s.Game.Hands != 2 {
		t.Fatal("not dealt after resuming")
	}
	h.ok(EndSession{Person: h.ids["a"]})
	h.mustLog("will end the session after this hand")
	h.play(holdem.Fold{})
	if !h.s.Ended() {
		t.Fatal("session should end with the hand")
	}
	if msg := h.refused(Rebuy{Person: h.ids["a"], Amount: 10}); msg != "the session has ended" {
		t.Fatal(msg)
	}
	h.checkInvariants()
	h.advance(time.Hour)
	if h.closed == nil || h.closed.Reason != "ended" {
		t.Fatalf("closed %+v", h.closed)
	}
}

func TestIdleClose(t *testing.T) {
	h := pair(t)
	h.advance(29 * time.Minute)
	h.ok(Chat{Person: h.ids["a"], Text: "hi"})
	h.advance(29 * time.Minute)
	if h.closed != nil {
		t.Fatal("chat should keep the table open")
	}
	h.advance(time.Minute)
	if h.closed == nil || h.closed.Reason != "idle" || h.closed.Message != "the table closed after 30 minutes idle" {
		t.Fatalf("closed %+v", h.closed)
	}
}

func TestBotsOnly(t *testing.T) {
	h := newHarness(t, DefaultSettings, "")
	h.join("host")
	h.ok(AddBot{Person: h.ids["host"], Seat: 1, BuyIn: 1000, Name: "bob", Level: "easy"})
	h.ok(AddBot{Person: h.ids["host"], Seat: 2, BuyIn: 1000, Name: "sue", Level: "medium"})
	h.refused(AddBot{Person: h.ids["host"], Seat: 3, BuyIn: 1000, Name: "Bob", Level: "easy"})
	h.ok(Start{Person: h.ids["host"]})
	for range 300 {
		h.advance(5 * time.Second)
		h.checkInvariants()
		if b := Ledger(h.s).Balance; b != 0 {
			t.Fatalf("balance %d", b)
		}
	}
	if h.s.Game.Hands < 20 {
		t.Fatalf("bots played only %d hands", h.s.Game.Hands)
	}
	h.ok(RemoveBot{Person: h.ids["host"], Seat: 1})
	h.advance(time.Minute)
	if h.s.Seats[1] != nil {
		t.Fatal("bot not removed")
	}
}

func TestViewsHideCards(t *testing.T) {
	h := pair(t)
	h.join("watcher")
	res := h.ok(Start{Person: h.ids["a"]})
	va := ViewFor(h.s, h.ids["a"], h.now)
	if va.You != 1 || va.Players[0].Cards == [2]holdem.Card{} || va.Players[1].Cards != [2]holdem.Card{} || !va.Legal.Turn {
		t.Fatalf("a's view %+v", va)
	}
	if vw := ViewFor(h.s, h.ids["watcher"], h.now); vw.Players[0].Cards != [2]holdem.Card{} || vw.Watchers[0] != "watcher" {
		t.Fatal("watcher sees cards")
	}
	for _, e := range EventsFor(h.s, res.Events, h.ids["watcher"]) {
		if _, ok := e.(holdem.Dealt); ok {
			t.Fatal("watcher got hole cards")
		}
	}
	h.play(holdem.Fold{})
	hb := EventsFor(h.s, h.events, h.ids["b"])
	done := hb[len(hb)-1].(HandDone)
	if done.Record.Players[0].Cards != nil || done.Record.Players[1].Cards == nil {
		t.Fatalf("b's record %+v", done.Record.Players)
	}
	if hands, more := HistoryFor(h.s, "a", 0); len(hands) != 1 || more || hands[0].Players[0].Cards == nil {
		t.Fatal("history page")
	}
}

// A table saved mid-hand and loaded again plays on exactly as the original.
func TestRoundTrip(t *testing.T) {
	h := newHarness(t, DefaultSettings, "pw")
	h.ok(Join{ID: "host", Name: "host", Password: "pw"})
	h.ids["host"] = "host"
	for i, name := range []string{"bob", "sue", "al"} {
		h.ok(AddBot{Person: "host", Seat: i + 1, BuyIn: 1000, Name: name, Level: "medium"})
	}
	h.ok(Start{Person: "host"})
	h.advance(37 * time.Second)

	b, err := json.Marshal(h.s)
	if err != nil {
		t.Fatal(err)
	}
	var loaded State
	if err := json.Unmarshal(b, &loaded); err != nil {
		t.Fatal(err)
	}
	copy := &harness{t: t, s: &loaded, now: h.now, ids: map[string]string{}, tokens: map[string]string{}}
	h.advance(5 * time.Minute)
	copy.advance(5 * time.Minute)
	if h.s.Game.Hands < 5 || !reflect.DeepEqual(stacksOf(h.s), stacksOf(copy.s)) || h.s.Game.Hands != copy.s.Game.Hands {
		t.Fatalf("hands %d vs %d, stacks %v vs %v", h.s.Game.Hands, copy.s.Game.Hands, stacksOf(h.s), stacksOf(copy.s))
	}
	if !checkPassword(loaded.Password, "pw") {
		t.Fatal("password lost")
	}
}

func stacksOf(s *State) []uint {
	var out []uint
	for _, p := range s.Game.Players {
		out = append(out, p.Stack)
	}
	return out
}

func TestRestore(t *testing.T) {
	h := pair(t)
	h.join("watcher")
	h.ok(Start{Person: h.ids["a"]})
	b, _ := json.Marshal(h.s)
	var loaded State
	json.Unmarshal(b, &loaded)
	h.s = &loaded
	h.now = h.now.Add(time.Hour)
	Restore(h.s, h.now)
	h.checkInvariants()
	if len(h.s.People) != 2 || h.s.People[h.ids["a"]].Connected {
		t.Fatal("seated people wait to reconnect, watchers are gone")
	}
	if until(h.s.Timers, TimerTurn, h.now) != 30000 {
		t.Fatal("the player to act gets a fresh turn")
	}
	res := h.do(Join{ID: "new", Name: "x", Token: h.tokens["a"]})
	if res.Joined == nil || res.Joined.ID != h.ids["a"] {
		t.Fatal("a reclaims their seat after the restart")
	}
	h.play(holdem.Call{})
	h.checkInvariants()
}
