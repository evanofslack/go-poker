package table

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// straddleTable seats names in seats 1, 2, 3... with straddles set to mode.
// The first hand puts the button on seat 1, so under the gun is seat 4.
func straddleTable(t *testing.T, mode string, names ...string) *harness {
	t.Helper()
	set := DefaultSettings
	set.Straddle = mode
	h := newHarness(t, set, "")
	for i, name := range names {
		h.join(name)
		h.sit(name, i+1, 1000)
	}
	return h
}

func (h *harness) asked() int {
	if a := h.s.StraddleAsk; a != nil {
		return a.Seat
	}
	return 0
}

func (h *harness) answer(name string, yes bool) {
	h.t.Helper()
	h.ok(Straddle{Person: h.ids[name], Yes: yes})
}

func TestStraddleChain(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d", "e")
	h.ok(Start{Person: h.ids["a"]})
	if h.s.Game.InHand() || h.asked() != 4 || h.s.StraddleAsk.Amount != 40 {
		t.Fatalf("the deal waits for d: in hand %v ask %+v", h.s.Game.InHand(), h.s.StraddleAsk)
	}
	if v := ViewFor(h.s, h.ids["e"], h.now); v.StraddleAsk.Seat != 4 || v.StraddleEndsIn != 5000 || v.StraddlePreset != PresetAsk {
		t.Fatalf("view %+v ends in %d preset %q", v.StraddleAsk, v.StraddleEndsIn, v.StraddlePreset)
	}
	if msg := h.refused(Straddle{Person: h.ids["e"], Yes: true}); msg != "you aren't being asked to straddle" {
		t.Fatal(msg)
	}
	h.answer("d", true)
	h.mustLog("d straddles 40")
	if h.asked() != 5 || h.s.StraddleAsk.Amount != 80 {
		t.Fatalf("e is asked to re-straddle: %+v", h.s.StraddleAsk)
	}
	h.answer("e", true)
	h.mustLog("e straddles 80")
	// the button can't straddle, so that's the last one and the hand deals
	g := &h.s.Game
	if !g.InHand() || !slices.Equal(g.Straddles, []int{4, 5}) || g.ToAct != 1 || h.s.StraddleAsk != nil {
		t.Fatalf("straddles %v to act %d ask %+v", g.Straddles, g.ToAct, h.s.StraddleAsk)
	}
	if _, ok := h.s.Timers[TimerStraddle]; ok {
		t.Fatal("straddle timer left over")
	}
	h.checkInvariants()
}

func TestStraddleNo(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d", "e")
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", true)
	h.answer("e", false)
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
}

func TestStraddleTimesOut(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.ok(Start{Person: h.ids["a"]})
	h.advance(4 * time.Second)
	if h.s.Game.InHand() {
		t.Fatal("dealt before d's time ran out")
	}
	h.advance(time.Second)
	if g := &h.s.Game; !g.InHand() || len(g.Straddles) != 0 {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
}

func TestStraddleBetweenHands(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.s.Settings.DecisionTime = 10000
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", false)
	for h.s.Game.InHand() {
		h.play(holdem.Fold{})
	}
	// the button moves to b, so under the gun is a, asked as the pause starts
	if h.asked() != 1 {
		t.Fatalf("asked %d, want 1", h.asked())
	}
	// the pause is up, but the deal waits for a
	h.advance(7 * time.Second)
	if h.s.Game.InHand() || !h.s.StraddleAsk.Waiting {
		t.Fatalf("dealt before a answered: ask %+v", h.s.StraddleAsk)
	}
	h.answer("a", true)
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{1}) || g.Hands != 2 {
		t.Fatalf("hand %d straddles %v", g.Hands, g.Straddles)
	}
	h.checkInvariants()
}

func TestStraddleAnswerBeforeThePause(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", false)
	for h.s.Game.InHand() {
		h.play(holdem.Fold{})
	}
	h.answer("a", true)
	if h.s.Game.InHand() {
		t.Fatal("dealt before the pause was up")
	}
	h.advance(5 * time.Second)
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{1}) {
		t.Fatalf("straddles %v", g.Straddles)
	}
}

func TestForcedStraddle(t *testing.T) {
	h := straddleTable(t, StraddleForced, "a", "b", "c", "d", "e")
	h.ok(Start{Person: h.ids["a"]})
	h.mustLog("d straddles 40")
	if h.asked() != 5 {
		t.Fatalf("e is asked to re-straddle, asked %d", h.asked())
	}
	h.answer("e", false)
	if g := &h.s.Game; !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("straddles %v", g.Straddles)
	}
}

func TestForcedStraddleBot(t *testing.T) {
	h := straddleTable(t, StraddleForced, "a", "b", "c")
	h.ok(AddBot{Person: h.ids["a"], Seat: 4, BuyIn: 1000, Name: "robo", Level: "easy"})
	h.ok(Start{Person: h.ids["a"]})
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
}

func TestStraddleSkips(t *testing.T) {
	// a bot under the gun ends the asking
	h := straddleTable(t, StraddleOptional, "a", "b", "c")
	h.ok(AddBot{Person: h.ids["a"], Seat: 4, BuyIn: 1000, Name: "robo", Level: "easy"})
	h.ok(Start{Person: h.ids["a"]})
	if g := &h.s.Game; !g.InHand() || len(g.Straddles) != 0 {
		t.Fatalf("bot: in hand %v straddles %v", g.InHand(), g.Straddles)
	}

	// and so does someone disconnected
	h = straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.ok(Disconnect{Person: h.ids["d"]})
	h.ok(Start{Person: h.ids["a"]})
	if g := &h.s.Game; !g.InHand() || len(g.Straddles) != 0 {
		t.Fatalf("disconnected: in hand %v straddles %v", g.InHand(), g.Straddles)
	}
}

func TestStraddlePresets(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d", "e")
	h.ok(StraddlePreset{Person: h.ids["d"], Preset: PresetAlways})
	h.ok(Start{Person: h.ids["a"]})
	h.mustLog("d straddles 40")
	if h.asked() != 5 {
		t.Fatalf("asked %d, want e", h.asked())
	}
	// setting a preset while asked answers
	h.ok(StraddlePreset{Person: h.ids["e"], Preset: PresetNever})
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
	if v := ViewFor(h.s, h.ids["e"], h.now); v.StraddlePreset != PresetNever {
		t.Fatalf("preset %q", v.StraddlePreset)
	}
	h.refused(StraddlePreset{Person: h.ids["e"], Preset: "sometimes"})

	// never doesn't get out of a forced straddle
	h = straddleTable(t, StraddleForced, "a", "b", "c", "d")
	h.ok(StraddlePreset{Person: h.ids["d"], Preset: PresetNever})
	h.ok(Start{Person: h.ids["a"]})
	if g := &h.s.Game; !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("forced: straddles %v", g.Straddles)
	}
}

func TestStraddleSeatsChange(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d", "e")
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", true)
	// d sits out, so e is under the gun now and the asking starts over
	h.ok(SitOut{Person: h.ids["d"]})
	h.mustLog("the straddles start over")
	if a := h.s.StraddleAsk; a.Seat != 5 || a.Amount != 40 || len(a.Yes) != 0 {
		t.Fatalf("ask %+v", a)
	}
	h.answer("e", true)
	if g := &h.s.Game; !slices.Equal(g.Straddles, []int{5}) {
		t.Fatalf("straddles %v", g.Straddles)
	}
}

func TestStraddlePauseAndOff(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.ok(Start{Person: h.ids["a"]})
	h.ok(Pause{Person: h.ids["a"], Pause: true})
	if h.s.StraddleAsk != nil {
		t.Fatal("pausing clears the asking")
	}
	h.ok(Pause{Person: h.ids["a"], Pause: false})
	if h.asked() != 4 {
		t.Fatalf("resuming asks again, asked %d", h.asked())
	}

	set := h.s.Settings
	set.Straddle = StraddleOff
	h.ok(UpdateSettings{Person: h.ids["a"], Settings: set})
	if h.s.StraddleAsk != nil {
		t.Fatal("turning straddles off clears the asking")
	}
	h.advance(5 * time.Second)
	if g := &h.s.Game; !g.InHand() || len(g.Straddles) != 0 {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
}

func TestStraddleRecords(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d")
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", true)
	if _, ok := h.s.Acts[4]; ok {
		t.Fatal("the straddle shows as an act")
	}
	h.play(holdem.Fold{})
	h.play(holdem.Fold{})
	h.play(holdem.Fold{})

	rec := h.s.History[0]
	var kinds []string
	for _, a := range rec.Actions {
		kinds = append(kinds, a.Kind)
	}
	if want := []string{"small blind", "big blind", "straddle", "fold", "fold", "fold"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("actions %v", kinds)
	}
	if rec.Actions[2].To != 40 || rec.Players[3].Net != 30 {
		t.Fatalf("straddle %+v net %d", rec.Actions[2], rec.Players[3].Net)
	}
	if d := h.s.Totals["d"]; d.VPIP != 0 || d.PFR != 0 || d.Aggressive != 0 {
		t.Fatalf("a straddle isn't voluntary or a raise: %+v", d)
	}
}

func TestStraddleRestore(t *testing.T) {
	h := straddleTable(t, StraddleOptional, "a", "b", "c", "d", "e")
	h.ok(Start{Person: h.ids["a"]})
	h.answer("d", true)
	b, _ := json.Marshal(h.s)
	var loaded State
	json.Unmarshal(b, &loaded)
	h.s = &loaded
	h.now = h.now.Add(time.Minute)
	Restore(h.s, h.now)
	if a := h.s.StraddleAsk; a == nil || a.Seat != 5 || !slices.Equal(a.Yes, []int{4}) {
		t.Fatalf("ask %+v", a)
	}
	h.advance(5 * time.Second)
	if g := &h.s.Game; !g.InHand() || !slices.Equal(g.Straddles, []int{4}) {
		t.Fatalf("in hand %v straddles %v", g.InHand(), g.Straddles)
	}
	h.checkInvariants()
}
