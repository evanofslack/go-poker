package table

import "strconv"

// Totals add up how someone has played every hand at the table, for
// stats like how often they put money in. Kept by name, like the ledger.
type Totals struct {
	Hands         int    `json:"hands"`         // dealt in
	Bombs         int    `json:"bombs"`         // of them bomb pots, which have no preflop to count for VPIP and PFR
	VPIP          int    `json:"vpip"`          // hands they chose to put money in preflop
	PFR           int    `json:"pfr"`           // hands they raised preflop
	ThreeBetSpots int    `json:"threeBetSpots"` // hands they acted preflop facing one raise
	ThreeBets     int    `json:"threeBets"`     // and re-raised
	Aggressive    int    `json:"aggressive"`    // bets and raises
	Calls         int    `json:"calls"`
	FacedBets     int    `json:"facedBets"`    // times they acted facing a bet after the flop
	FoldedToBets  int    `json:"foldedToBets"` // and folded
	SawFlop       int    `json:"sawFlop"`      // hands still in when the flop came
	Showdowns     int    `json:"showdowns"`    // hands they showed
	ShowdownWins  int    `json:"showdownWins"` // and won a pot
	Net           int    `json:"net"`          // chips won or lost
	BestHand      string `json:"bestHand,omitempty"`
	BestScore     int    `json:"bestScore,omitempty"` // of BestHand, lower is better
}

// tally adds a finished hand to everyone's totals.
func tally(totals map[string]*Totals, r *HandRecord) {
	count := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	won := map[string]bool{}
	for _, pot := range r.Pots {
		for _, w := range pot.Winners {
			won[w] = true
		}
	}

	// preflop, once a hand each
	vpip, pfr, spot, threeBet, foldedPreflop := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	raises := 0
	var street string
	var top uint
	bets := map[string]uint{} // what each player has bet this street
	for _, a := range r.Actions {
		if a.Street != street {
			street, top = a.Street, 0
			clear(bets)
		}
		if street != "preflop" && a.Kind != "bomb" && a.Kind != "ante" && top > bets[a.Name] {
			s := totalsFor(totals, a.Name)
			s.FacedBets++
			s.FoldedToBets += count(a.Kind == "fold")
		}
		if a.Amount > 0 {
			bets[a.Name] = a.To
			top = max(top, a.To)
		}
		if a.Kind == "ante" || a.Kind == "small blind" || a.Kind == "big blind" || a.Kind == "straddle" || a.Kind == "bomb" {
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

	for _, p := range r.Players {
		s := totalsFor(totals, p.Name)
		s.Hands++
		s.Bombs += count(r.Bomb > 0)
		s.Net += p.Net
		s.VPIP += count(vpip[p.Name])
		s.PFR += count(pfr[p.Name])
		s.ThreeBetSpots += count(spot[p.Name])
		s.ThreeBets += count(threeBet[p.Name])
		s.SawFlop += count(len(r.Board) >= 3 && !foldedPreflop[p.Name])
		if p.Showdown {
			s.Showdowns++
			s.ShowdownWins += count(won[p.Name])
		}
		// totals are public, so only hands that were shown
		if p.Shown && p.Hand != "" {
			if s.BestHand == "" || p.Score < s.BestScore {
				s.BestHand, s.BestScore = p.Hand, p.Score
			}
		}
	}
}

func totalsFor(totals map[string]*Totals, name string) *Totals {
	s, ok := totals[name]
	if !ok {
		s = &Totals{}
		totals[name] = s
	}
	return s
}

// TotalsView copies the totals for sending.
func TotalsView(s *State) map[string]Totals {
	v := make(map[string]Totals, len(s.Totals))
	for name, t := range s.Totals {
		v[name] = *t
	}
	return v
}

func itoa(n uint) string { return strconv.FormatUint(uint64(n), 10) }
