package holdem

import "slices"

// Sit adds a player to seat with stack chips. They're dealt in from the next
// hand.
func (s *State) Sit(seat int, stack uint) error {
	if seat < 1 {
		return ErrNoSeat
	}
	if s.Player(seat) != nil {
		return ErrSeatTaken
	}
	i, _ := slices.BinarySearchFunc(s.Players, seat, func(p Player, seat int) int { return p.Seat - seat })
	s.Players = slices.Insert(s.Players, i, Player{Seat: seat, Stack: stack})
	return nil
}

// Leave removes the player in seat. Only between hands, since their chips may
// be in a pot.
func (s *State) Leave(seat int) error {
	if s.Player(seat) == nil {
		return ErrNoSeat
	}
	if s.InHand() {
		return ErrInHand
	}
	s.Players = slices.DeleteFunc(s.Players, func(p Player) bool { return p.Seat == seat })
	// the last result stays up, without the player who left
	for i := range s.Pots {
		pot := &s.Pots[i]
		pot.Eligible = slices.DeleteFunc(pot.Eligible, func(n int) bool { return n == seat })
		pot.Winners = slices.DeleteFunc(pot.Winners, func(n int) bool { return n == seat })
		for k := range pot.Runs {
			pot.Runs[k].Winners = slices.DeleteFunc(pot.Runs[k].Winners, func(n int) bool { return n == seat })
		}
	}
	return nil
}

// playerOutOfHand is the player in seat if they aren't in the hand.
func (s *State) playerOutOfHand(seat int) (*Player, error) {
	p := s.Player(seat)
	switch {
	case p == nil:
		return nil, ErrNoSeat
	case p.In || p.Committed > 0:
		return nil, ErrInHand
	}
	return p, nil
}

// AddChips adds n chips to the stack of the player in seat, while they aren't
// in a hand.
func (s *State) AddChips(seat int, n uint) error {
	p, err := s.playerOutOfHand(seat)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBadAmount
	}
	p.Stack += n
	return nil
}

// RemoveChips takes n chips from the stack of the player in seat, while they
// aren't in a hand.
func (s *State) RemoveChips(seat int, n uint) error {
	p, err := s.playerOutOfHand(seat)
	if err != nil {
		return err
	}
	if n == 0 || n > p.Stack {
		return ErrBadAmount
	}
	p.Stack -= n
	return nil
}

// SetSittingOut sits the player in seat out of the next hands, or back in.
// Sitting back in needs chips.
func (s *State) SetSittingOut(seat int, out bool) error {
	p, err := s.playerOutOfHand(seat)
	if err != nil {
		return err
	}
	if !out && p.Stack == 0 {
		return ErrBadAmount
	}
	p.SittingOut = out
	return nil
}
