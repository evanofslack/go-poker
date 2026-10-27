package table

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// runItTable seats names in seats 1, 2, 3... on a table that runs it. The
// first hand puts the button on seat 1, who acts first preflop.
func runItTable(t *testing.T, names ...string) *harness {
	t.Helper()
	set := DefaultSettings
	set.RunIt = true
	h := newHarness(t, set, "")
	for i, name := range names {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	h.ok(Start{Person: h.ids[names[0]]})
	return h
}

// shove has the first to act go all in and the next calls take it, the rest
// folding.
func (h *harness) shove(calls int) {
	h.t.Helper()
	h.play(holdem.Raise{To: h.s.Game.Legal(h.s.Game.ToAct).RaiseMax})
	for range calls {
		h.play(holdem.Call{})
	}
	for h.s.Game.Betting {
		h.play(holdem.Fold{})
	}
}

func (h *harness) pick(name string, runs int) {
	h.t.Helper()
	h.ok(RunIt{Person: h.ids[name], Runs: runs})
}

func TestRunItTwice(t *testing.T) {
	h := runItTable(t, "a", "b")
	h.shove(1)
	v := h.s.RunVote
	if v == nil || v.Max != 4 || len(v.Seats) != 2 {
		t.Fatalf("vote %+v", v)
	}
	if _, ok := h.s.Timers[TimerStreet]; ok {
		t.Fatal("the flop waits for the vote")
	}
	if view := ViewFor(h.s, h.ids["b"], h.now); view.RunVote == nil || view.RunVoteEndsIn != 5000 {
		t.Fatalf("view vote %+v ends in %d", view.RunVote, view.RunVoteEndsIn)
	}
	h.pick("a", 2)
	if h.s.RunVote == nil {
		t.Fatal("the vote waits for b")
	}
	h.pick("b", 3)
	h.mustLog("running it 2 times")
	if h.s.RunVote != nil || h.s.Game.Runs != 2 {
		t.Fatalf("vote %+v runs %d", h.s.RunVote, h.s.Game.Runs)
	}
	h.advance(time.Minute)
	if len(h.s.History) == 0 {
		t.Fatal("the hand should be over")
	}
	rec := h.s.History[0]
	if len(rec.Boards) != 2 || len(rec.Board) != 5 || len(rec.Pots) != 2 || rec.Pots[1].Run != 1 {
		t.Fatalf("record boards %v board %v pots %+v", rec.Boards, rec.Board, rec.Pots)
	}
	var won uint
	for _, n := range h.s.Stats[0].Won {
		won += n
	}
	if won != 2000 {
		t.Fatalf("stats won %v", h.s.Stats[0].Won)
	}
}

func TestRunItOnceEndsTheVote(t *testing.T) {
	h := runItTable(t, "a", "b")
	h.shove(1)
	h.pick("b", 1)
	h.mustLog("running it once")
	if h.s.RunVote != nil || h.s.Game.Runs != 1 {
		t.Fatalf("vote %+v runs %d", h.s.RunVote, h.s.Game.Runs)
	}
	h.advance(time.Minute)
	if rec := h.s.History[0]; len(rec.Boards) != 0 || len(rec.Pots) != 1 {
		t.Fatalf("record boards %v pots %+v", rec.Boards, rec.Pots)
	}
}

func TestRunItCovered(t *testing.T) {
	// a covers b, so once b calls all in, a has chips left but nobody to bet with
	h := newHarness(t, DefaultSettings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 2000)
	h.sit("b", 2, 1000)
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Raise{To: 2000})
	h.play(holdem.Call{})
	if v := h.s.RunVote; v == nil || len(v.Seats) != 2 {
		t.Fatalf("vote %+v", v)
	}
	h.pick("a", 2)
	h.pick("b", 2)
	h.mustLog("running it 2 times")
}

func TestRunItSmallestPick(t *testing.T) {
	h := runItTable(t, "a", "b", "c")
	h.shove(2)
	h.pick("a", 3)
	h.pick("b", 4)
	h.pick("c", 2)
	h.mustLog("running it 2 times")
}

func TestRunItTimeout(t *testing.T) {
	h := runItTable(t, "a", "b")
	h.shove(1)
	h.pick("a", 3)
	h.advance(5 * time.Second)
	h.mustLog("running it once")
	if h.s.Game.Runs != 1 {
		t.Fatalf("runs %d", h.s.Game.Runs)
	}
}

func TestRunItVoters(t *testing.T) {
	h := runItTable(t, "a", "b", "c")
	h.shove(1) // c folds
	h.join("w")
	for name, want := range map[string]string{
		"c": "only players in the hand vote",
		"w": "only players in the hand vote",
	} {
		if msg := h.refused(RunIt{Person: h.ids[name], Runs: 2}); msg != want {
			t.Fatalf("%s: %s", name, msg)
		}
	}
	if msg := h.refused(RunIt{Person: h.ids["a"], Runs: 5}); msg != "pick from 1 to 4 runs" {
		t.Fatal(msg)
	}
	h.pick("a", 2)
	h.pick("b", 2)
	h.mustLog("running it 2 times")
	if msg := h.refused(RunIt{Person: h.ids["a"], Runs: 2}); msg != "there's no run it vote" {
		t.Fatal(msg)
	}
}

func TestRunItDisconnectedPicksOnce(t *testing.T) {
	h := runItTable(t, "a", "b")
	h.play(holdem.Raise{To: h.s.Game.Legal(1).RaiseMax})
	h.do(Disconnect{Person: h.ids["a"]})
	h.play(holdem.Call{})
	h.mustLog("running it once")
}

func TestRunItBotsPickTheMost(t *testing.T) {
	set := DefaultSettings
	set.RunIt = true
	h := newHarness(t, set, "")
	h.join("a")
	h.sit("a", 1, 1000)
	h.ok(AddBot{Person: h.ids["a"], Seat: 2, BuyIn: 1000, Name: "bot", Level: "easy"})
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Raise{To: h.s.Game.Legal(1).RaiseMax})
	for h.s.Game.Betting {
		h.advance(time.Second)
	}
	if h.s.Game.Player(2).In {
		if v := h.s.RunVote; v == nil || v.Picks[2] != 4 {
			t.Fatalf("vote %+v", v)
		}
	}
}

func TestRunItOff(t *testing.T) {
	set := DefaultSettings
	set.RunIt = false
	h := newHarness(t, set, "")
	for i, name := range []string{"a", "b"} {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	h.ok(Start{Person: h.ids["a"]})
	h.shove(1)
	if h.s.RunVote != nil {
		t.Fatal("no vote with run it off")
	}
	h.advance(time.Minute)
	if len(h.s.History) == 0 || len(h.s.History[0].Boards) != 0 {
		t.Fatal("the hand runs out once")
	}
}

func TestRunItRestore(t *testing.T) {
	h := runItTable(t, "a", "b")
	h.shove(1)
	h.pick("a", 2)
	b, _ := json.Marshal(h.s)
	var loaded State
	json.Unmarshal(b, &loaded)
	h.s = &loaded
	h.now = h.now.Add(time.Minute)
	Restore(h.s, h.now)
	if v := h.s.RunVote; v == nil || v.Picks[1] != 2 {
		t.Fatalf("vote %+v", v)
	}
	if at := h.s.Timers[TimerRunVote]; !at.Equal(h.now.Add(5 * time.Second)) {
		t.Fatalf("vote timer at %v", at)
	}
	h.checkInvariants()
}
