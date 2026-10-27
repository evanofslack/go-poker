package table

import (
	"maps"
	"slices"
	"time"
)

// MaxRuns caps how many times a table runs the board out.
const MaxRuns = 4

// RunVote asks the players still in a hand how many times to run the rest of
// the board out, once everyone is all in, or all but one, with cards to come. The smallest pick wins, so anyone can keep
// it to once.
type RunVote struct {
	Seats []int       `json:"seats"` // still in the hand, who vote
	Picks map[int]int `json:"picks"` // by seat, once they answer
	Max   int         `json:"max"`   // the most runs on offer
}

// askRuns starts the vote when betting is over for the hand, with cards to
// come, on a table that runs it. Bots pick the most, so the people decide,
// and anyone disconnected picks once. It reports whether the hand waits for
// the vote.
func (r *reducer) askRuns() bool {
	s := r.s
	n := min(MaxRuns, s.Game.MaxRuns())
	if !s.Settings.RunIt || s.RunVote != nil || n < 2 {
		return false
	}
	v := &RunVote{Picks: map[int]int{}, Max: n}
	for _, p := range s.Game.Players {
		if !p.In {
			continue
		}
		v.Seats = append(v.Seats, p.Seat)
		seat := s.Seats[p.Seat]
		if person := s.People[seat.Person]; seat.Bot != "" {
			v.Picks[p.Seat] = n
		} else if person == nil || !person.Connected {
			v.Picks[p.Seat] = 1
		}
	}
	s.RunVote = v
	if r.countRuns() {
		return true
	}
	r.schedule(TimerRunVote, time.Duration(s.Settings.DecisionTime)*time.Millisecond)
	return true
}

// runIt is the person's pick in the vote.
func (r *reducer) runIt(p *Person, runs int) error {
	v := r.s.RunVote
	switch {
	case v == nil:
		return fail("there's no run it vote")
	case p.Seat == 0 || !slices.Contains(v.Seats, p.Seat):
		return fail("only players in the hand vote")
	case runs < 1 || runs > v.Max:
		return fail("pick from 1 to %d runs", v.Max)
	}
	v.Picks[p.Seat] = runs
	r.countRuns()
	return nil
}

// countRuns ends the vote once everyone has picked, or someone picked once,
// and reports whether it did.
func (r *reducer) countRuns() bool {
	v := r.s.RunVote
	if len(v.Picks) < len(v.Seats) && !slices.Contains(slices.Collect(maps.Values(v.Picks)), 1) {
		return false
	}
	r.endRunVote()
	return true
}

// endRunVote runs the board out the smallest number of times picked, with
// anyone who didn't pick counting as once, and deals on.
func (r *reducer) endRunVote() {
	s := r.s
	v := s.RunVote
	runs := v.Max
	for _, n := range v.Seats {
		pick, ok := v.Picks[n]
		if !ok {
			pick = 1
		}
		runs = min(runs, pick)
	}
	s.RunVote = nil
	r.cancel(TimerRunVote)
	if s.Game.SetRuns(runs) != nil {
		return
	}
	if runs == 1 {
		r.log("running it once")
	} else {
		r.log("running it %d times", runs)
	}
	r.schedule(TimerStreet, s.Timing.StreetDelay)
}
