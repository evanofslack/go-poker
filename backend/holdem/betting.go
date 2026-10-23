package holdem

// Action is a player's move on their turn: Check, Call, Raise or Fold.
type Action interface {
	action()
}

type Check struct{}
type Call struct{}
type Fold struct{}

// Raise bets or raises to To, the player's whole bet this street.
type Raise struct {
	To uint `json:"to"`
}

func (Check) action() {}
func (Call) action()  {}
func (Fold) action()  {}
func (Raise) action() {}

// Legal is what a player may do now. All zero when it isn't their turn.
type Legal struct {
	Turn  bool `json:"turn"`
	Check bool `json:"check"`
	Call  uint `json:"call"` // chips to call, 0 when checking; less than owed when all in
	// RaiseMin and RaiseMax bound a bet or raise, as the whole bet this street.
	// Both 0 when the player can't raise. A short stack can only go all in, at
	// RaiseMax, below RaiseMin.
	RaiseMin uint `json:"raiseMin"`
	RaiseMax uint `json:"raiseMax"`
}

// Legal says what the player in seat may do now.
func (s *State) Legal(seat int) Legal {
	p := s.Player(seat)
	if !s.Betting || s.ToAct != seat || p == nil {
		return Legal{}
	}
	top := s.topBet()
	owe := top - p.Bet
	l := Legal{Turn: true, Check: owe == 0, Call: min(owe, p.Stack)}
	// a short all-in doesn't reopen betting for someone who already acted
	if p.Stack > owe && !p.Acted {
		l.RaiseMax = p.Bet + p.Stack
		l.RaiseMin = min(top+s.MinRaise, l.RaiseMax)
	}
	return l
}

// Act makes the move a for the player in seat, on their turn.
func (s *State) Act(seat int, a Action) ([]Event, error) {
	l := s.Legal(seat)
	if !l.Turn {
		return nil, ErrNotYourTurn
	}
	p := s.Player(seat)
	top := s.topBet()
	e := Acted{Seat: seat, Street: s.Street}

	switch a := a.(type) {
	case Check:
		if !l.Check {
			return nil, ErrIllegal
		}
		e.Kind = KindCheck
	case Call:
		if l.Check {
			return nil, ErrIllegal
		}
		e.Kind = KindCall
		e.Amount = p.put(l.Call)
	case Raise:
		allIn := a.To == l.RaiseMax
		if l.RaiseMax == 0 || a.To > l.RaiseMax || a.To <= top || (a.To < l.RaiseMin && !allIn) {
			return nil, ErrIllegal
		}
		e.Kind = KindRaise
		if top == 0 {
			e.Kind = KindBet
		}
		// a full raise reopens betting for everyone who already acted
		if a.To-top >= s.MinRaise {
			s.MinRaise = a.To - top
			for i := range s.Players {
				s.Players[i].Acted = false
			}
		}
		e.Amount = p.put(a.To - p.Bet)
		s.Aggressor = seat
	case Fold:
		e.Kind = KindFold
		p.In = false
	default:
		return nil, ErrIllegal
	}
	p.Acted = true
	e.To = p.Bet
	e.AllIn = p.allIn()

	return append([]Event{e}, s.advance()...), nil
}

// needsToAct reports whether p still has a decision this street.
func (s *State) needsToAct(p *Player) bool {
	return p.canBet() && (!p.Acted || p.Bet < s.topBet())
}

// advance moves the action on after a blind or a move: to the next player
// with a decision, or on to the end of the street or hand.
func (s *State) advance() []Event {
	s.updatePots()
	if s.count(func(p *Player) bool { return p.In }) < 2 {
		return s.foldWin()
	}
	next := s.next(s.ToAct, s.needsToAct)
	// one player left with chips who has matched the top bet has nobody to bet against
	alone := s.count((*Player).canBet) < 2 && next != 0 && s.Player(next).Bet >= s.topBet()
	if next != 0 && !alone {
		s.ToAct = next
		return nil
	}
	return s.closeStreet()
}

// closeStreet ends the betting on a street. Without two players left who can
// bet, the board runs out one NextStreet at a time with every hand face up.
func (s *State) closeStreet() []Event {
	s.Betting = false
	s.ToAct = 0
	var events []Event
	if s.count((*Player).canBet) < 2 {
		events = append(events, s.returnUncalled()...)
		for i := range s.Players {
			if p := &s.Players[i]; p.In && !p.Shown {
				p.Shown = true
				events = append(events, Shown{p.Seat, p.Cards})
			}
		}
	}
	s.updatePots()
	if s.Street == River {
		events = append(events, s.showdown()...)
	}
	return events
}

// returnUncalled gives back the part of the biggest bet nobody else matched.
// Chips from players who folded count as matched: they stay in the pot.
func (s *State) returnUncalled() []Event {
	var top, second uint
	var bettor *Player
	for i := range s.Players {
		p := &s.Players[i]
		switch {
		case p.Committed > top:
			second, top, bettor = top, p.Committed, p
		case p.Committed > second:
			second = p.Committed
		}
	}
	if bettor == nil || !bettor.In || top == second {
		return nil
	}
	back := top - second
	bettor.Stack += back
	bettor.Committed -= back
	bettor.Bet -= min(back, bettor.Bet)
	return []Event{Returned{bettor.Seat, back}}
}
