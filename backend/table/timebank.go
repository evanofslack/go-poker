package table

import "time"

// The time bank: each player holds time cards, each adding Settings.TimeCard
// to their turn timer. They sit down with Settings.TimeBank and earn one every
// Settings.TimeEvery hands dealt in, up to Settings.TimeMax. A card is spent
// with AddTime, or by itself when a connected player's turn runs out. Bots
// never hold any.

// addTime spends one of p's time cards on their turn.
func (r *reducer) addTime(p *Person) error {
	s := r.s
	_, timed := s.Timers[TimerTurn]
	switch {
	case p.Seat == 0:
		return fail("take a seat first")
	case !s.Game.Betting || s.Game.ToAct != p.Seat || !timed:
		return fail("you can only add time on your turn")
	case s.Seats[p.Seat].TimeCards == 0:
		return fail("you have no time cards left")
	}
	r.spendTime(p.Seat)
	r.log("%s adds %ds", p.Name, s.Settings.TimeCard/1000)
	return nil
}

// autoTime spends a time card for the player in seat when their turn runs out,
// if they're connected and have one. It reports whether it did.
func (r *reducer) autoTime(seat int) bool {
	s := r.s
	st := s.Seats[seat]
	if st == nil || st.TimeCards == 0 {
		return false
	}
	if p := s.People[st.Person]; p == nil || !p.Connected {
		return false
	}
	s.Timers[TimerTurn] = r.now // spendTime adds to the deadline that just passed
	r.spendTime(seat)
	r.log("%s's time bank adds %ds", st.Name, s.Settings.TimeCard/1000)
	return true
}

// spendTime moves the turn timer back by a card's worth and takes the card.
func (r *reducer) spendTime(seat int) {
	s := r.s
	s.Seats[seat].TimeCards--
	s.Timers[TimerTurn] = s.Timers[TimerTurn].Add(time.Duration(s.Settings.TimeCard) * time.Millisecond)
}

// earnTime counts the hand just played toward a time card for everyone dealt
// into it.
func (r *reducer) earnTime() {
	s := r.s
	if s.Settings.TimeMax == 0 || s.Settings.TimeEvery == 0 {
		return
	}
	for n, st := range s.Seats {
		if p := s.Game.Player(n); st.Bot != "" || p == nil || !p.Dealt {
			continue
		}
		st.TimeHands++
		if st.TimeHands >= s.Settings.TimeEvery {
			st.TimeHands = 0
			st.TimeCards = min(st.TimeCards+1, s.Settings.TimeMax)
		}
	}
}
