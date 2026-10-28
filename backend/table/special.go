package table

// specialHand makes the next hand a special one, or calls it off with kind
// "". Mid-hand it's the hand after this one.
func (r *reducer) specialHand(p *Person, kind string) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case kind == "" && s.SpecialNext == "":
		return fail("the next hand isn't special")
	case kind == "":
		r.log("%s called off the %s", p.Name, specialNames[s.SpecialNext])
		s.SpecialNext = ""
		return nil
	case specialNames[kind] == "":
		return fail("that isn't a special hand")
	case isBomb(kind) && s.Settings.BombPot == 0:
		return fail("bomb pots are off at this table")
	case isDouble(kind) && !s.Settings.fitsDouble():
		return fail("there aren't enough cards for two boards at %d seats", s.Settings.Seats)
	case s.SpecialNext == kind:
		return fail("the next hand is already a %s", specialNames[kind])
	}
	s.SpecialNext = kind
	if isBomb(kind) {
		r.log("%s makes the next hand a %s of %d each", p.Name, specialNames[kind], s.Settings.BombPot)
	} else {
		r.log("%s makes the next hand %s", p.Name, specialNames[kind])
	}
	return nil
}

// specialDue is how many hands apart set's schedule deals special hands: an
// orbit is a hand for each player dealt in.
func (s *State) specialDue(set Settings) int {
	n := int(set.SpecialEvery)
	if set.SpecialUnit == UnitOrbits {
		n *= max(s.Game.ReadyCount(), 2)
	}
	return n
}

// scheduleSpecial counts a normal hand towards the schedule, and makes the
// next hand special once due-1 have gone by. A special the host called
// stands, and the count waits for it.
func (r *reducer) scheduleSpecial() {
	s := r.s
	if !s.Settings.scheduled() {
		return
	}
	if !s.special() {
		s.SpecialHands++
	}
	if s.SpecialNext == "" && s.SpecialHands >= s.specialDue(s.Settings)-1 {
		s.SpecialNext = s.Settings.Special
		s.SpecialHands = 0
	}
}

// special is whether the hand being played, or the last one, was special.
func (s *State) special() bool { return s.Game.Bomb > 0 || s.Game.Double }

// specialIn is how many hands until the schedule's next special hand, the
// one it deals included, or 0 with no schedule. A schedule the host changed
// mid-hand counts already.
func (s *State) specialIn() int {
	set := s.Settings
	if s.Next != nil {
		set = *s.Next
	}
	switch {
	case !set.scheduled():
		return 0
	case s.SpecialNext != "":
		return 1
	case s.Game.InHand() && !s.special():
		return max(s.specialDue(set)-s.SpecialHands-1, 1)
	}
	return max(s.specialDue(set)-s.SpecialHands, 1)
}
