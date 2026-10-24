package table

import "slices"

// KindStandUp is the stand-up game, as a side game kind.
const KindStandUp = "stand-up"

// maxStandUpBB is the most a stand-up payout can be, in big blinds.
const maxStandUpBB = 20

// StandUp is a round of the stand-up game. Everyone dealt into its first
// hand starts standing, and sits down by winning or splitting a pot with
// their cards face up. The last one left standing pays everyone who sat down.
type StandUp struct {
	Payout  uint     `json:"payout"`  // from the last one standing to each player who sat
	Pending bool     `json:"pending"` // starts with the next hand
	Players []string `json:"players"` // names dealt into its first hand
	Sat     []string `json:"sat"`
}

// Standing are the round's players who haven't sat down yet.
func (u *StandUp) Standing() []string {
	return slices.DeleteFunc(slices.Clone(u.Players), func(n string) bool { return slices.Contains(u.Sat, n) })
}

// seatOf is the seat of the player called name, 0 if they have none or are
// leaving it.
func seatOf(s *State, name string) int {
	for n, seat := range s.Seats {
		if seat.Name == name && !seat.Leaving {
			return n
		}
	}
	return 0
}

// wonPot reports whether the player in seat won or split a pot of the hand
// just ended.
func wonPot(s *State, seat int) bool {
	p := s.Game.Player(seat)
	if p == nil || !p.Dealt || s.Game.InHand() {
		return false
	}
	for _, pot := range s.Game.Pots {
		if slices.Contains(pot.Winners, seat) {
			return true
		}
	}
	return false
}

// sitsDown reports whether the player in seat, called name, sits down in the
// stand-up round by showing their cards now.
func sitsDown(s *State, seat int, name string) bool {
	u := s.StandUp
	return u != nil && !u.Pending && slices.Contains(u.Standing(), name) && wonPot(s, seat)
}

func (r *reducer) startStandUp(p *Person, payout uint) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case s.StandUp != nil:
		return fail("a stand-up round is already going")
	case payout == 0 || payout > maxStandUpBB*s.Settings.BigBlind:
		return fail("the payout must be between 1 chip and %d big blinds", maxStandUpBB)
	}
	s.StandUp = &StandUp{Payout: payout, Pending: true}
	s.LastStandUp = 0
	r.log("%s started a stand-up round for %d each, from the next hand", p.Name, payout)
	return nil
}

// cancelStandUp ends the round without a payout. With no round going, it
// dismisses running the last one again.
func (r *reducer) cancelStandUp(p *Person) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case s.StandUp == nil && s.LastStandUp == 0:
		return fail("there's no stand-up round going")
	case s.StandUp == nil:
		s.LastStandUp = 0
		return nil
	}
	s.StandUp = nil
	r.log("%s cancelled the stand-up round", p.Name)
	return nil
}

// beginStandUp starts a pending round with the players dealt into this hand.
func (r *reducer) beginStandUp(seats []int) {
	u := r.s.StandUp
	if u == nil || !u.Pending {
		return
	}
	u.Pending = false
	for _, n := range seats {
		u.Players = append(u.Players, r.s.Seats[n].Name)
	}
	r.log("stand-up: %s standing, %d each", plural(len(u.Players), "player"), u.Payout)
}

// standUp sits down the round's players who won a pot of the hand just ended
// and showed, then pays out if that leaves one standing.
func (r *reducer) standUp() []Payment {
	s := r.s
	u := s.StandUp
	if u == nil || u.Pending || s.Game.InHand() {
		return nil
	}
	for _, name := range u.Standing() {
		seat := seatOf(s, name)
		if seat != 0 && sitsDown(s, seat, name) && s.Game.Player(seat).Shown {
			u.Sat = append(u.Sat, name)
			r.log("%s sits down", name)
		}
	}

	standing := u.Standing()
	var paid []Payment
	switch len(standing) {
	case 0:
		r.log("the last ones standing sat down together: the stand-up round ends without a payout")
	case 1:
		paid = r.payStandUp(standing[0], "is the last one standing")
	default:
		return nil
	}
	s.LastStandUp = u.Payout
	s.StandUp = nil
	return paid
}

// payStandUp has name pay the round's payout to each player who sat down
// and is still seated, as far as their chips go.
func (r *reducer) payStandUp(name, why string) []Payment {
	s := r.s
	u := s.StandUp
	from := 0
	for n, seat := range s.Seats {
		if seat.Name == name {
			from = n
		}
	}
	var paid []Payment
	var total uint
	for _, n := range u.Sat {
		to := seatOf(s, n)
		if to == 0 {
			continue
		}
		if pm, ok := r.pay(KindStandUp, from, to, u.Payout); ok {
			paid = append(paid, pm)
			total += pm.Amount
		}
	}
	r.log("%s %s and pays %d to %s", name, why, total, plural(len(paid), "player"))
	if len(paid) > 0 {
		r.res.Events = append(r.res.Events, Paid{paid})
	}
	return paid
}

// leaveStandUp has a player leaving while standing pay as if they lost. If
// that leaves one standing, the round ends without them paying.
func (r *reducer) leaveStandUp(name string) {
	s := r.s
	u := s.StandUp
	if u == nil || u.Pending || !slices.Contains(u.Standing(), name) {
		return
	}
	paid := r.payStandUp(name, "left while standing")
	u.Players = slices.DeleteFunc(u.Players, func(n string) bool { return n == name })
	if standing := u.Standing(); len(standing) <= 1 {
		if len(standing) == 1 {
			r.log("%s is the last one standing: the stand-up round ends without them paying", standing[0])
		}
		s.LastStandUp = u.Payout
		s.StandUp = nil
	}
	if n := len(s.History); n > 0 && len(paid) > 0 {
		h := &s.History[n-1]
		r.latePayments(h, paid)
		r.res.Events = append(r.res.Events, HandUpdated{*h})
	}
}
