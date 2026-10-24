package table

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

func TestAntesRecorded(t *testing.T) {
	settings := DefaultSettings
	settings.Ante = 5
	h := newHarness(t, settings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Call{})
	h.play(holdem.Check{})
	h.advance(time.Second)
	h.play(holdem.Raise{To: 100})
	h.play(holdem.Fold{})

	rec := h.s.History[0]
	want := []string{"ante", "ante", "small blind", "big blind", "call", "check", "bet", "fold"}
	var kinds []string
	for _, a := range rec.Actions {
		kinds = append(kinds, a.Kind)
	}
	if !reflect.DeepEqual(kinds, want) || rec.Ante != 5 || rec.Players[1].Net != 25 {
		t.Fatalf("record %+v", rec)
	}
	// posting an ante isn't voluntary
	if h.s.Totals["a"].VPIP != 1 || h.s.Totals["b"].VPIP != 0 {
		t.Fatalf("totals a %+v b %+v", h.s.Totals["a"], h.s.Totals["b"])
	}
}

func TestShowAfterHand(t *testing.T) {
	h := pair(t)
	h.join("watcher")
	h.ok(Start{Person: h.ids["a"]})
	h.refused(Show{Person: h.ids["b"]})
	h.play(holdem.Fold{})

	h.refused(Show{Person: h.ids["watcher"]})
	res := h.ok(Show{Person: h.ids["b"]})
	if !h.s.History[0].Players[1].Shown {
		t.Fatal("the record should have b's cards shown")
	}
	var updated *HandUpdated
	for _, e := range EventsFor(h.s, res.Events, h.ids["watcher"]) {
		if u, ok := e.(HandUpdated); ok {
			updated = &u
		}
	}
	if updated == nil || updated.Record.Players[1].Cards == nil || updated.Record.Players[0].Cards != nil {
		t.Fatalf("the watcher should see b's cards and not a's: %+v", updated)
	}
	if v := ViewFor(h.s, h.ids["watcher"], h.now); v.Players[1].Cards == [2]holdem.Card{} {
		t.Fatal("b's cards should be face up")
	}
	h.refused(Show{Person: h.ids["b"]})
	// a folded, and can show too
	h.ok(Show{Person: h.ids["a"]})
}

func TestMuckedShowdown(t *testing.T) {
	h := pair(t)
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Call{})
	h.play(holdem.Check{})
	for range 3 {
		h.advance(time.Second)
		h.play(holdem.Check{})
		h.play(holdem.Check{})
	}
	if h.s.Game.InHand() {
		t.Fatal("the hand should be over")
	}

	rec := h.s.History[0]
	winners := rec.Pots[0].Winners
	for _, p := range rec.Players {
		if !p.Showdown || h.s.Totals[p.Name].Showdowns != 1 {
			t.Fatalf("%s reached showdown: %+v", p.Name, p)
		}
		won := slices.Contains(winners, p.Name)
		if p.Shown != won {
			t.Fatalf("%s won %v but shown %v", p.Name, won, p.Shown)
		}
		if !won {
			if h.s.Totals[p.Name].BestHand != "" {
				t.Fatal("a mucked hand counted as a best hand")
			}
			other := rec.Players[0].Name
			if other == p.Name {
				other = rec.Players[1].Name
			}
			if hands, _ := HistoryFor(h.s, other, 0); hands[0].Players[p.Seat-1].Cards != nil {
				t.Fatal("a mucked hand reached the other player")
			}
		}
	}
}

func TestDuckHunt(t *testing.T) {
	h := pair(t)
	h.join("watcher")
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Fold{})

	h.refused(Hunt{Person: h.ids["watcher"]})
	h.advance(4 * time.Second)
	h.ok(Hunt{Person: h.ids["a"]})
	if v := ViewFor(h.s, h.ids["watcher"], h.now); len(v.Hunted) != 5 || len(v.Board) != 0 {
		t.Fatalf("hunted %v board %v", v.Hunted, v.Board)
	}
	h.mustLog("a went duck hunting")
	h.refused(Hunt{Person: h.ids["b"]})

	// the hunted cards stay up a while before the next hand
	h.advance(2 * time.Second)
	if h.s.Game.InHand() {
		t.Fatal("dealt over the hunted cards")
	}
	h.advance(time.Second)
	if h.s.Game.Hands != 2 || h.s.Game.Hunted != nil {
		t.Fatal("the next hand should be dealt, without the hunt")
	}
}

func TestDuckHuntOff(t *testing.T) {
	settings := DefaultSettings
	settings.DuckHunt = false
	h := newHarness(t, settings, "")
	h.join("a")
	h.join("b")
	h.sit("a", 1, 1000)
	h.sit("b", 2, 1000)
	h.ok(Start{Person: h.ids["a"]})
	h.play(holdem.Fold{})
	h.refused(Hunt{Person: h.ids["a"]})
}
