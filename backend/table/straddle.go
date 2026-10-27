package table

import (
	"slices"
	"time"
)

// Straddle presets: how a seated player answers straddle asks.
const (
	PresetAsk    = "ask"
	PresetAlways = "always"
	PresetNever  = "never"
)

// StraddleAsk is the straddles for the next hand, asked one seat at a time
// from under the gun while the table waits between hands. Each yes asks the
// next seat to double it; a no, the timer, or running out of seats ends it.
type StraddleAsk struct {
	Seat    int   `json:"seat"`    // being asked, 0 once the asking is over
	Amount  uint  `json:"amount"`  // what Seat would straddle
	Asked   []int `json:"asked"`   // seats asked so far, in order, Seat included
	Yes     []int `json:"yes"`     // seats straddling, in order
	Waiting bool  `json:"waiting"` // the deal is due and waits for the answer
}

func (s *State) straddling() bool {
	return s.Settings.Straddle == StraddleOptional || s.Settings.Straddle == StraddleForced
}

// askStraddles keeps the ask in step with the table, after every command: it
// starts one when a hand is coming, starts over if the seats changed under
// it, and drops it when no hand is.
func (r *reducer) askStraddles() {
	s := r.s
	if !s.straddling() || !s.Running || s.Paused || s.Ended() || s.Game.InHand() {
		r.dropStraddles()
		return
	}
	pos := s.Game.Positions().Straddle
	a := s.StraddleAsk
	if a != nil && slices.Equal(pos[:min(len(a.Asked), len(pos))], a.Asked) &&
		!(a.Seat == 0 && len(a.Asked) == len(a.Yes) && len(pos) > len(a.Asked)) {
		// the seats asked are still the first who can straddle, and if
		// everyone said yes, nobody new can
		return
	}
	if a != nil && len(a.Yes) > 0 {
		r.log("the seats changed, so the straddles start over")
	}
	s.StraddleAsk = &StraddleAsk{Asked: []int{}, Yes: []int{}, Waiting: a != nil && a.Waiting}
	r.quiet = false
	r.askNext()
}

// askNext asks the next seat that can straddle. Forced straddles and players
// who always straddle say yes without being asked; bots, players who never
// do, and anyone disconnected end the asking at their seat.
func (r *reducer) askNext() {
	s := r.s
	a := s.StraddleAsk
	pos := s.Game.Positions().Straddle
	for len(a.Asked) < len(pos) {
		k := len(a.Asked)
		n := pos[k]
		a.Asked = append(a.Asked, n)
		seat := s.Seats[n]
		amount := s.Game.Config.BigBlind << (k + 1)
		if !(k == 0 && s.Settings.Straddle == StraddleForced) && seat.Straddle != PresetAlways {
			person := s.People[seat.Person]
			if seat.Bot != "" || person == nil || !person.Connected || seat.Straddle == PresetNever {
				break
			}
			a.Seat, a.Amount = n, amount
			r.schedule(TimerStraddle, time.Duration(s.Settings.DecisionTime)*time.Millisecond)
			return
		}
		a.Yes = append(a.Yes, n)
		r.log("%s straddles %d", seat.Name, amount)
	}
	r.endStraddles()
}

// endStraddles stops asking, and deals if the hand was waiting for it.
func (r *reducer) endStraddles() {
	a := r.s.StraddleAsk
	if a == nil {
		return
	}
	a.Seat, a.Amount = 0, 0
	r.cancel(TimerStraddle)
	if a.Waiting {
		a.Waiting = false
		r.dealHand(true)
	}
}

func (r *reducer) dropStraddles() {
	if r.s.StraddleAsk != nil {
		r.s.StraddleAsk = nil
		r.quiet = false
	}
	r.cancel(TimerStraddle)
}

// straddle answers the ask for the person being asked.
func (r *reducer) straddle(p *Person, yes bool) error {
	a := r.s.StraddleAsk
	if a == nil || a.Seat == 0 || a.Seat != p.Seat {
		return fail("you aren't being asked to straddle")
	}
	if !yes {
		r.endStraddles()
		return nil
	}
	a.Yes = append(a.Yes, a.Seat)
	r.log("%s straddles %d", p.Name, a.Amount)
	a.Seat, a.Amount = 0, 0
	r.cancel(TimerStraddle)
	r.askNext()
	return nil
}

// straddlePreset sets how the person answers straddle asks, answering one
// they're being asked now.
func (r *reducer) straddlePreset(p *Person, preset string) error {
	if p.Seat == 0 {
		return fail("take a seat first")
	}
	switch preset {
	case PresetAsk, PresetAlways, PresetNever:
	default:
		return fail("unknown straddle preset")
	}
	r.s.Seats[p.Seat].Straddle = preset
	if a := r.s.StraddleAsk; a != nil && a.Seat == p.Seat && preset != PresetAsk {
		return r.straddle(p, preset == PresetAlways)
	}
	return nil
}
