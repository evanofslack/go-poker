package server

import (
	"encoding/json"
	"log/slog"

	"github.com/evanofslack/go-poker/poker"
)

// maxHands caps how many hands of stats a table keeps.
const maxHands = 1000

// handStats is a snapshot taken as a hand ends. Stats are worked out from
// these on the client.
type handStats struct {
	Hand   int             `json:"hand"`   // counts from 1
	Stacks map[string]uint `json:"stacks"` // everyone seated, after the pots are paid
	Played []string        `json:"played"` // names dealt in
	Won    map[string]uint `json:"won"`    // pots each winner took, their own chips included
}

// recordHand snapshots the hand that just ended and sends it to everyone.
// Caller holds t.mu.
func recordHand(t *table, view *poker.GameView) {
	h := handStats{
		Hand:   t.handCount + 1,
		Stacks: make(map[string]uint, len(view.Players)),
		Played: t.current.played(),
		Won:    make(map[string]uint),
	}
	t.handCount++
	for _, p := range view.Players {
		h.Stacks[p.Username] = p.Stack
	}
	for _, pot := range view.Pots {
		if len(pot.WinningPlayerNums) == 0 {
			continue
		}
		// odd chips aren't tracked; this is close enough for stats
		share := pot.Amt / uint(len(pot.WinningPlayerNums))
		for _, pn := range pot.WinningPlayerNums {
			h.Won[view.Players[pn].Username] += share
		}
	}
	t.hands = append(t.hands, h)
	if len(t.hands) > maxHands {
		t.hands = t.hands[len(t.hands)-maxHands:]
	}
	t.send(createStats([]handStats{h}, false))
}

// createStats sends hands: every hand kept when full, else new ones to add.
func createStats(hands []handStats, full bool) []byte {
	resp, err := json.Marshal(updateStats{base{actionUpdateStats}, hands, full})
	if err != nil {
		slog.Default().Warn("Marshal stats", "error", err)
	}
	return resp
}
