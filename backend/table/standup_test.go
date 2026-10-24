package table

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// standUpTable seats names in seats 1, 2, 3... with 1000 each, the first
// hosting, and starts a stand-up round for 40 and the game.
func standUpTable(t *testing.T, names ...string) *harness {
	h := newHarness(t, DefaultSettings, "")
	for i, name := range names {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	h.refused(StartStandUp{Person: h.ids[names[1]], Payout: 40})
	h.ok(StartStandUp{Person: h.ids[names[0]], Payout: 40})
	h.ok(Start{Person: h.ids[names[0]]})
	if u := h.s.StandUp; u == nil || u.Pending || !slices.Equal(u.Players, names) {
		t.Fatalf("round %+v", u)
	}
	return h
}

// nextHand waits out the pause between hands.
func (h *harness) nextHand() {
	h.t.Helper()
	hand := h.s.Game.Hands
	h.advance(10 * time.Second)
	if h.s.Game.Hands != hand+1 {
		h.t.Fatalf("hand %d wasn't dealt: %s", hand+1, h)
	}
}

// bWinsShowdown plays a three handed hand to a showdown b wins: a is on the
// button, b and c in the blinds.
func bWinsShowdown(h *harness) {
	h.s.Game.Player(1).Cards = hole("3c 4d")
	h.s.Game.Player(2).Cards = hole("Ah Ad")
	h.s.Game.Player(3).Cards = hole("5c 6d")
	h.play(holdem.Call{})
	h.play(holdem.Call{})
	h.checkDown("Kh Kd 9s 8c 2h")
}

func TestStandUpRound(t *testing.T) {
	h := standUpTable(t, "a", "b", "c")
	bWinsShowdown(h)
	if u := h.s.StandUp; !slices.Equal(u.Sat, []string{"b"}) {
		t.Fatalf("b won at showdown and sits: %+v", u)
	}
	h.mustLog("b sits down")

	// c wins without a call, and has to show to sit
	h.nextHand()
	h.play(holdem.Fold{})         // b, after the big blind
	h.play(holdem.Raise{To: 100}) // c
	h.play(holdem.Fold{})         // a
	if got := ViewFor(h.s, h.ids["c"], h.now).ShowClaims; !slices.Equal(got, []string{"stand-up"}) {
		t.Fatalf("c's claims %v", got)
	}
	if len(h.s.StandUp.Sat) != 1 {
		t.Fatal("c sat without showing")
	}
	a, b, c := h.stack("a"), h.stack("b"), h.stack("c")
	h.ok(Show{Person: h.ids["c"]})

	// a is left standing and pays both
	if h.stack("a") != a-80 || h.stack("b") != b+40 || h.stack("c") != c+40 {
		t.Fatalf("stacks a %d b %d c %d", h.stack("a"), h.stack("b"), h.stack("c"))
	}
	h.mustLog("a is the last one standing and pays 80 to 2 players")
	if h.s.StandUp != nil || h.s.LastStandUp != 40 {
		t.Fatalf("the round should end, ready to run again: %+v %d", h.s.StandUp, h.s.LastStandUp)
	}
	if rec := h.s.History[1]; len(rec.Payments) != 2 || rec.Payments[0].Kind != KindStandUp {
		t.Fatalf("payments %+v", rec.Payments)
	}

	// dismissing run it again
	h.ok(CancelStandUp{Person: h.ids["a"]})
	if h.s.LastStandUp != 0 {
		t.Fatal("not dismissed")
	}
}

func TestStandUpLateJoinerLeftOut(t *testing.T) {
	h := standUpTable(t, "a", "b")
	h.join("d")
	h.sit("d", 3, 1000)
	h.play(holdem.Fold{})
	h.nextHand()
	if u := h.s.StandUp; slices.Contains(u.Players, "d") {
		t.Fatalf("d joined mid-round: %+v", u)
	}
	if got := ViewFor(h.s, h.ids["d"], h.now).ShowClaims; slices.Contains(got, "stand-up") {
		t.Fatal("d can't sit in this round")
	}
}

func TestStandUpSplitEndsWithoutPayout(t *testing.T) {
	h := standUpTable(t, "a", "b")
	h.s.Game.Player(1).Cards = hole("2c 3d")
	h.s.Game.Player(2).Cards = hole("2d 3c")
	h.play(holdem.Call{})
	h.checkDown("Ah Kh Qh Jh Th") // the board plays: both split
	if h.s.StandUp != nil || h.stack("a") != 1000 || h.stack("b") != 1000 {
		t.Fatalf("round %+v, stacks a %d b %d", h.s.StandUp, h.stack("a"), h.stack("b"))
	}
	h.mustLog("ends without a payout")
}

func TestStandUpLeaveWhileStanding(t *testing.T) {
	h := standUpTable(t, "a", "b", "c")
	bWinsShowdown(h)
	b := h.stack("b")
	h.ok(LeaveSeat{Person: h.ids["a"]})
	if h.stack("b") != b+40 {
		t.Fatalf("a should pay b on leaving: %d", h.stack("b"))
	}
	h.mustLog("a left while standing and pays 40 to 1 player")
	h.mustLog("c is the last one standing: the stand-up round ends without them paying")
	if h.s.StandUp != nil || h.stack("c") != 980 {
		t.Fatalf("round %+v, c %d", h.s.StandUp, h.stack("c"))
	}
	if entry := h.s.Ledger[len(h.s.Ledger)-1]; entry.Name != "a" || entry.Amount != 940 {
		t.Fatalf("a cashes out after paying: %+v", entry)
	}
}

func TestStandUpCancel(t *testing.T) {
	h := standUpTable(t, "a", "b")
	h.refused(StartStandUp{Person: h.ids["a"], Payout: 40})
	h.refused(CancelStandUp{Person: h.ids["b"]})
	h.ok(CancelStandUp{Person: h.ids["a"]})
	if h.s.StandUp != nil || h.s.LastStandUp != 0 {
		t.Fatal("a cancelled round doesn't pay or offer a rerun")
	}
	// a pending round sends empty lists, not nulls
	h.ok(StartStandUp{Person: h.ids["a"], Payout: 40})
	if b, _ := json.Marshal(ViewFor(h.s, h.ids["a"], h.now).StandUp); !strings.Contains(string(b), `"players":[]`) {
		t.Fatalf("pending round %s", b)
	}
	h.refused(StartStandUp{Person: h.ids["a"], Payout: 20*maxStandUpBB + 1})
}
