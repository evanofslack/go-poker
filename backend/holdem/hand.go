package holdem

import (
	"encoding/binary"
	"math/rand/v2"
	"slices"
)

// shuffle shuffles deck for hand, from seed. A var so tests can stack the deck.
var shuffle = func(deck []Card, seed [32]byte, hand int) []Card {
	binary.LittleEndian.PutUint64(seed[24:], binary.LittleEndian.Uint64(seed[24:])^uint64(hand))
	rand.New(rand.NewChaCha8(seed)).Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// draw takes n cards off the top of the deck.
func (s *State) draw(n int) []Card {
	cards := slices.Clone(s.Deck[len(s.Deck)-n:])
	s.Deck = s.Deck[:len(s.Deck)-n]
	return cards
}

// Positions is where the button, blinds and straddles would go in the next
// hand, without changing anything. The button moves to the next player who
// isn't sitting out and has chips; heads up, the button posts the small blind.
// Straddles go from under the gun up to the seat before the button, each
// double the last, and stop at the first player who can't put theirs in and
// still have chips behind after the ante. Zero without two players to deal in.
func (s *State) Positions() Positions {
	if s.ReadyCount() < 2 {
		return Positions{}
	}
	ready := (*Player).ready
	var pos Positions
	pos.Button = s.next(s.Button, ready)
	pos.SB = s.next(pos.Button, ready)
	if s.ReadyCount() == 2 {
		pos.SB = pos.Button
	}
	pos.BB = s.next(pos.SB, ready)
	for seat := s.next(pos.BB, ready); seat != pos.Button; seat = s.next(seat, ready) {
		p := s.Player(seat)
		if p.Stack-min(p.Stack, s.Config.Ante) <= s.straddle(len(pos.Straddle)) {
			break
		}
		pos.Straddle = append(pos.Straddle, seat)
	}
	return pos
}

// straddle is the k-th straddle, from 0: two big blinds, then double each time.
func (s *State) straddle(k int) uint { return s.Config.BigBlind << (k + 1) }

// StartHand deals a hand to every player who isn't sitting out and has chips,
// with the first straddles of Positions().Straddle straddling. The button
// moves to the next of them; heads up, the button posts the small blind and
// acts first preflop.
func (s *State) StartHand(straddles int) ([]Event, error) {
	if s.InHand() {
		return nil, ErrInHand
	}
	if s.ReadyCount() < 2 {
		return nil, ErrNotEnoughPlayers
	}
	pos := s.Positions()
	if straddles < 0 || straddles > len(pos.Straddle) {
		return nil, ErrIllegal
	}

	for i := range s.Players {
		p := &s.Players[i]
		*p = Player{Seat: p.Seat, Stack: p.Stack, SittingOut: p.SittingOut}
	}
	s.Hands++
	s.Board = nil
	s.Hunted = nil
	s.Runs, s.RunFrom, s.Boards = 0, 0, nil
	s.Pots = nil
	s.Aggressor = 0
	s.Deck = shuffle(Deck(s.Config.Game), s.Seed, s.Hands)
	s.Street = Preflop
	s.Betting = true
	s.MinRaise = s.Config.BigBlind << straddles
	s.Button, s.SB, s.BB = pos.Button, pos.SB, pos.BB
	s.Straddles = slices.Clone(pos.Straddle[:straddles])

	started := HandStarted{Hand: s.Hands, Button: s.Button, SB: s.SB, BB: s.BB}
	var dealt []Event
	for i := range s.Players {
		if p := &s.Players[i]; p.ready() {
			p.In, p.Dealt = true, true
			p.Cards = s.draw(s.Config.Game.HoleCards())
			started.Seats = append(started.Seats, p.Seat)
			dealt = append(dealt, Dealt{p.Seat, p.Cards})
		}
	}
	events := append([]Event{started}, dealt...)
	if s.Config.Ante > 0 {
		for _, seat := range started.Seats {
			p := s.Player(seat)
			amt := p.putDead(s.Config.Ante)
			events = append(events, Acted{Seat: seat, Street: Preflop, Kind: KindAnte, Amount: amt, AllIn: p.allIn()})
		}
	}
	type blind struct {
		seat int
		kind string
		amt  uint
	}
	blinds := []blind{{s.SB, KindSmallBlind, s.Config.SmallBlind}, {s.BB, KindBigBlind, s.Config.BigBlind}}
	for k, seat := range s.Straddles {
		blinds = append(blinds, blind{seat, KindStraddle, s.straddle(k)})
	}
	for _, b := range blinds {
		p := s.Player(b.seat)
		amt := p.put(b.amt)
		events = append(events, Acted{Seat: b.seat, Street: Preflop, Kind: b.kind, Amount: amt, To: p.Bet, AllIn: p.allIn()})
	}

	// action starts after the big blind or the last straddle, who gets the
	// option since they haven't acted
	s.ToAct = blinds[len(blinds)-1].seat
	return append(events, s.advance()...), nil
}

// MaxRuns is how many times the deck lets the rest of the board run out, or
// 0 when SetRuns isn't legal. There are no burn cards, so each run takes only
// the cards still to come.
func (s *State) MaxRuns() int {
	if !s.InHand() || s.Betting || s.count((*Player).canBet) >= 2 || len(s.Board) == 5 || s.Runs != 0 {
		return 0
	}
	return len(s.Deck) / (5 - len(s.Board))
}

// SetRuns runs the rest of the board out n times, once nobody can bet before
// the river. Each pot splits between the runs. Only once a hand.
func (s *State) SetRuns(n int) error {
	if n < 1 || n > s.MaxRuns() {
		return ErrIllegal
	}
	s.Runs, s.RunFrom = n, len(s.Board)
	return nil
}

// runsLeft is how many runs of the board are still to deal.
func (s *State) runsLeft() int { return max(s.Runs, 1) - len(s.Boards) }

// NextStreet deals the next street once betting on the last has closed. On
// the river, with the board running out more than once, it deals the next run.
func (s *State) NextStreet() ([]Event, error) {
	if !s.InHand() || s.Betting {
		return nil, ErrIllegal
	}
	if s.Street == River {
		return s.nextRun()
	}
	for i := range s.Players {
		s.Players[i].Bet = 0
		s.Players[i].Acted = false
	}
	s.MinRaise = s.Config.BigBlind
	s.Aggressor = 0

	n := 1
	switch s.Street {
	case Preflop:
		s.Street, n = Flop, 3
	case Flop:
		s.Street = Turn
	case Turn:
		s.Street = River
	}
	cards := s.draw(n)
	s.Board = append(s.Board, cards...)
	events := []Event{StreetDealt{Street: s.Street, Cards: cards}}
	if s.Street == River {
		s.Boards = append(s.Boards, slices.Clone(s.Board))
	}

	if s.count((*Player).canBet) >= 2 {
		s.Betting = true
		s.ToAct = s.Button
		return append(events, s.advance()...), nil
	}
	if s.Street == River && s.runsLeft() == 0 {
		events = append(events, s.showdown()...)
	}
	return events, nil
}

// nextRun deals another whole run of the board: the cards before the runs
// split, then fresh ones. After the last, the showdown.
func (s *State) nextRun() ([]Event, error) {
	if s.runsLeft() <= 0 {
		return nil, ErrIllegal
	}
	cards := s.draw(5 - s.RunFrom)
	s.Boards = append(s.Boards, append(slices.Clone(s.Board[:s.RunFrom]), cards...))
	events := []Event{StreetDealt{Street: River, Cards: cards, Run: len(s.Boards) - 1}}
	if s.runsLeft() == 0 {
		events = append(events, s.showdown()...)
	}
	return events, nil
}

// updatePots splits the chips put in this hand into a main pot and a side pot
// for each all-in amount.
func (s *State) updatePots() {
	var levels []uint
	for _, p := range s.Players {
		if p.allIn() {
			levels = append(levels, p.Committed)
		}
	}
	var top uint
	for _, p := range s.Players {
		top = max(top, p.Committed)
	}
	levels = append(levels, top)
	slices.Sort(levels)
	levels = slices.Compact(levels)

	s.Pots = nil
	var prev uint
	for _, level := range levels {
		pot := Pot{Eligible: []int{}}
		for _, p := range s.Players {
			pot.Amount += min(p.Committed, level) - min(p.Committed, prev)
			if p.In && p.Committed >= level {
				pot.Eligible = append(pot.Eligible, p.Seat)
			}
		}
		prev = level
		if pot.Amount == 0 {
			continue
		}
		// dead money above everyone still in goes to the last pot they can win
		if len(pot.Eligible) == 0 && len(s.Pots) > 0 {
			s.Pots[len(s.Pots)-1].Amount += pot.Amount
			continue
		}
		s.Pots = append(s.Pots, pot)
	}
}

// foldWin gives every pot to the one player left.
func (s *State) foldWin() []Event {
	s.Betting = false
	s.ToAct = 0
	winner := s.next(0, func(p *Player) bool { return p.In })
	var events []Event
	for i := range s.Pots {
		pot := &s.Pots[i]
		pot.Winners = []int{winner}
		s.Player(winner).Stack += pot.Amount
		events = append(events, PotWon{Pot: i, Amount: pot.Amount, Winners: pot.Winners})
	}
	return append(events, s.endHand()...)
}

// showdown pays each pot to its best hand on each board. A pot splits evenly
// between the boards, odd chips to the earlier runs, and each run's share
// between its winners, odd chips to the first winner left of the button.
// Winners show, and so does the last to bet or raise on the river, who was
// called; other hands are mucked.
func (s *State) showdown() []Event {
	s.Betting = false
	s.ToAct = 0

	show := []int{s.Aggressor}
	n := uint(len(s.Boards))
	for i := range s.Pots {
		pot := &s.Pots[i]
		pot.Runs, pot.Winners = nil, nil
		for k, b := range s.Boards {
			run := PotRun{Amount: pot.Amount / n, Score: worstScore + 1}
			if uint(k) < pot.Amount%n {
				run.Amount++
			}
			for _, seat := range pot.Eligible {
				cards, score := s.Config.Game.Best(s.Player(seat).Cards, b)
				switch {
				case score < run.Score:
					run.Score, run.Cards, run.Winners = score, cards, []int{seat}
				case score == run.Score:
					run.Winners = append(run.Winners, seat)
				}
			}
			run.Hand = HandName(run.Score, run.Cards)
			pot.Runs = append(pot.Runs, run)
			show = append(show, run.Winners...)
		}
		// everyone who won any run, by seat
		for _, p := range s.Players {
			if slices.ContainsFunc(pot.Runs, func(r PotRun) bool { return slices.Contains(r.Winners, p.Seat) }) {
				pot.Winners = append(pot.Winners, p.Seat)
			}
		}
	}

	var events []Event
	for i := range s.Players {
		if p := &s.Players[i]; p.In && !p.Shown && slices.Contains(show, p.Seat) {
			p.Shown = true
			events = append(events, Shown{p.Seat, p.Cards})
		}
	}
	for i, pot := range s.Pots {
		for k, run := range pot.Runs {
			share := run.Amount / uint(len(run.Winners))
			for _, seat := range run.Winners {
				s.Player(seat).Stack += share
			}
			odd := s.next(s.Button, func(p *Player) bool { return slices.Contains(run.Winners, p.Seat) })
			s.Player(odd).Stack += run.Amount % uint(len(run.Winners))
			events = append(events, PotWon{Pot: i, Run: k, Amount: run.Amount, Winners: run.Winners, Hand: run.Hand})
		}
	}
	return append(events, s.endHand()...)
}

// endHand clears the hand, keeping the cards, boards and pots up to show the
// result, and the deck for Hunt. Busted players sit out.
func (s *State) endHand() []Event {
	s.Street = NoStreet
	s.Betting = false
	s.ToAct = 0
	for i := range s.Players {
		p := &s.Players[i]
		p.Bet, p.Committed, p.In, p.Acted = 0, 0, false, false
		if p.Stack == 0 {
			p.SittingOut = true
		}
	}
	return []Event{HandEnded{s.Hands}}
}

// Show turns up the cards of the player in seat, between hands, if they were
// dealt into the last one.
func (s *State) Show(seat int) ([]Event, error) {
	p := s.Player(seat)
	switch {
	case p == nil:
		return nil, ErrNoSeat
	case s.InHand():
		return nil, ErrInHand
	case !p.Dealt || p.Shown || len(p.Cards) == 0:
		return nil, ErrIllegal
	}
	p.Shown = true
	return []Event{Shown{p.Seat, p.Cards}}, nil
}

// Hunt turns up the rest of the board after a hand that ended before the
// river: the cards that would have come, dealt street by street from the
// same deck. They count for nothing.
func (s *State) Hunt() ([]Event, error) {
	if s.InHand() || len(s.Board) == 5 || s.Hunted != nil || len(s.Deck) < 5-len(s.Board) {
		return nil, ErrIllegal
	}
	var cards []Card
	for len(s.Board)+len(cards) < 5 {
		n := 1
		if len(s.Board)+len(cards) == 0 {
			n = 3
		}
		cards = append(cards, s.draw(n)...)
	}
	s.Hunted = cards
	return []Event{Hunted{cards}}, nil
}
