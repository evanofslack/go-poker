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
	Hand    int             `json:"hand"`    // counts from 1
	Started int64           `json:"started"` // unix ms, 0 if unknown
	Stacks  map[string]uint `json:"stacks"`  // everyone seated, after the pots are paid
	Played  []string        `json:"played"`  // names dealt in
	Won     map[string]uint `json:"won"`     // pots each winner took, their own chips included
}

// playerTotals add up how someone has played every hand at the table, for
// stats like how often they put money in. Kept by name, like the ledger.
type playerTotals struct {
	Hands         int    `json:"hands"`         // dealt in
	VPIP          int    `json:"vpip"`          // hands they chose to put money in preflop
	PFR           int    `json:"pfr"`           // hands they raised preflop
	ThreeBetSpots int    `json:"threeBetSpots"` // hands they acted preflop facing one raise
	ThreeBets     int    `json:"threeBets"`     // and re-raised
	Aggressive    int    `json:"aggressive"`    // bets and raises
	Calls         int    `json:"calls"`
	SawFlop       int    `json:"sawFlop"`      // hands still in when the flop came
	Showdowns     int    `json:"showdowns"`    // hands they showed
	ShowdownWins  int    `json:"showdownWins"` // and won a pot
	Net           int    `json:"net"`          // chips won or lost
	BestHand      string `json:"bestHand,omitempty"`
	bestScore     int    // of BestHand, lower is better
}

// tally adds a finished hand to everyone's totals.
func tally(totals map[string]*playerTotals, r *handRecord) {
	won := map[string]bool{}
	for _, pot := range r.Pots {
		for _, w := range pot.Winners {
			won[w] = true
		}
	}

	// preflop, once a hand each
	vpip, pfr, spot, threeBet, foldedPreflop := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	raises := 0
	for _, a := range r.Actions {
		if a.Kind == "small blind" || a.Kind == "big blind" {
			continue
		}
		if a.Street == "preflop" {
			if raises == 1 {
				spot[a.Name] = true
				threeBet[a.Name] = threeBet[a.Name] || a.Kind == "raise"
			}
			switch a.Kind {
			case "call", "bet":
				vpip[a.Name] = true
			case "raise":
				vpip[a.Name], pfr[a.Name] = true, true
				raises++
			case "fold":
				foldedPreflop[a.Name] = true
			}
		}
		s := totalsFor(totals, a.Name)
		switch a.Kind {
		case "bet", "raise":
			s.Aggressive++
		case "call":
			s.Calls++
		}
	}

	count := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	for _, p := range r.Players {
		s := totalsFor(totals, p.Name)
		s.Hands++
		s.Net += p.Net
		s.VPIP += count(vpip[p.Name])
		s.PFR += count(pfr[p.Name])
		s.ThreeBetSpots += count(spot[p.Name])
		s.ThreeBets += count(threeBet[p.Name])
		s.SawFlop += count(len(r.Board) >= 3 && !foldedPreflop[p.Name])
		if p.Shown {
			s.Showdowns++
			s.ShowdownWins += count(won[p.Name])
			if s.BestHand == "" || p.score < s.bestScore {
				s.BestHand, s.bestScore = p.Hand, p.score
			}
		}
	}
}

func totalsFor(totals map[string]*playerTotals, name string) *playerTotals {
	s, ok := totals[name]
	if !ok {
		s = &playerTotals{}
		totals[name] = s
	}
	return s
}

// totalsView copies the totals for sending.
func (t *table) totalsView() map[string]playerTotals {
	v := make(map[string]playerTotals, len(t.totals))
	for name, s := range t.totals {
		v[name] = *s
	}
	return v
}

// recordHand snapshots the hand that just ended, r as played, and sends it to
// everyone with the totals. Caller holds t.mu.
func recordHand(t *table, view *poker.GameView, r *handRecord) {
	h := handStats{
		Hand:   t.handCount + 1,
		Stacks: make(map[string]uint, len(view.Players)),
		Played: r.played(),
		Won:    make(map[string]uint),
	}
	if r != nil {
		h.Started = r.Started
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
	t.send(createStats([]handStats{h}, t.totalsView(), false))
}

// createStats sends hands, every hand kept when full, else new ones to add,
// and everyone's totals.
func createStats(hands []handStats, totals map[string]playerTotals, full bool) []byte {
	resp, err := json.Marshal(updateStats{base{actionUpdateStats}, hands, full, totals})
	if err != nil {
		slog.Default().Warn("Marshal stats", "error", err)
	}
	return resp
}
