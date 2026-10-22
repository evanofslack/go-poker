package holdem

import (
	"encoding/binary"
	"math/rand/v2"
	"slices"

	"github.com/alexclewontin/riverboat/eval"
)

// shuffle is hand's deck, from seed. A var so tests can stack the deck.
var shuffle = func(seed [32]byte, hand int) []Card {
	binary.LittleEndian.PutUint64(seed[24:], binary.LittleEndian.Uint64(seed[24:])^uint64(hand))
	deck := slices.Clone(eval.DefaultDeck)
	rand.New(rand.NewChaCha8(seed)).Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// draw takes n cards off the top of the deck.
func (s *State) draw(n int) []Card {
	cards := slices.Clone(s.Deck[len(s.Deck)-n:])
	s.Deck = s.Deck[:len(s.Deck)-n]
	return cards
}

// StartHand deals a hand to every player who isn't sitting out and has chips.
// The button moves to the next of them; heads up, the button posts the small
// blind and acts first preflop.
func (s *State) StartHand() ([]Event, error) {
	if s.InHand() {
		return nil, ErrInHand
	}
	if s.ReadyCount() < 2 {
		return nil, ErrNotEnoughPlayers
	}

	for i := range s.Players {
		p := &s.Players[i]
		*p = Player{Seat: p.Seat, Stack: p.Stack, SittingOut: p.SittingOut}
	}
	s.Hands++
	s.Board = nil
	s.Pots = nil
	s.Deck = shuffle(s.Seed, s.Hands)
	s.Street = Preflop
	s.Betting = true
	s.MinRaise = s.Config.BigBlind

	ready := (*Player).ready
	s.Button = s.next(s.Button, ready)
	s.SB = s.next(s.Button, ready)
	if s.ReadyCount() == 2 {
		s.SB = s.Button
	}
	s.BB = s.next(s.SB, ready)

	started := HandStarted{Hand: s.Hands, Button: s.Button, SB: s.SB, BB: s.BB}
	var dealt []Event
	for i := range s.Players {
		if p := &s.Players[i]; p.ready() {
			p.In, p.Dealt = true, true
			copy(p.Cards[:], s.draw(2))
			started.Seats = append(started.Seats, p.Seat)
			dealt = append(dealt, Dealt{p.Seat, p.Cards})
		}
	}
	events := append([]Event{started}, dealt...)
	for _, b := range []struct {
		seat int
		kind string
		amt  uint
	}{{s.SB, KindSmallBlind, s.Config.SmallBlind}, {s.BB, KindBigBlind, s.Config.BigBlind}} {
		p := s.Player(b.seat)
		amt := p.put(b.amt)
		events = append(events, Acted{Seat: b.seat, Street: Preflop, Kind: b.kind, Amount: amt, To: p.Bet, AllIn: p.allIn()})
	}

	// action starts after the big blind
	s.ToAct = s.BB
	return append(events, s.advance()...), nil
}

// NextStreet deals the next street once betting on the last has closed.
func (s *State) NextStreet() ([]Event, error) {
	if !s.InHand() || s.Betting {
		return nil, ErrIllegal
	}
	for i := range s.Players {
		s.Players[i].Bet = 0
		s.Players[i].Acted = false
	}
	s.MinRaise = s.Config.BigBlind

	n := 1
	switch s.Street {
	case Preflop:
		s.Street, n = Flop, 3
	case Flop:
		s.Street = Turn
	case Turn:
		s.Street = River
	default:
		return nil, ErrIllegal
	}
	cards := s.draw(n)
	s.Board = append(s.Board, cards...)
	events := []Event{StreetDealt{s.Street, cards}}

	if s.count((*Player).canBet) >= 2 {
		s.Betting = true
		s.ToAct = s.Button
		return append(events, s.advance()...), nil
	}
	if s.Street == River {
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

// showdown turns up every hand still in and pays each pot to its best hand.
// Split pots share evenly, with odd chips to the first winner left of the
// button.
func (s *State) showdown() []Event {
	s.Betting = false
	s.ToAct = 0
	var events []Event
	for i := range s.Players {
		if p := &s.Players[i]; p.In && !p.Shown {
			p.Shown = true
			events = append(events, Shown{p.Seat, p.Cards})
		}
	}

	b := s.Board
	for i := range s.Pots {
		pot := &s.Pots[i]
		pot.Score = worstScore + 1
		for _, seat := range pot.Eligible {
			c := s.Player(seat).Cards
			cards, score := eval.BestFiveOfSeven(c[0], c[1], b[0], b[1], b[2], b[3], b[4])
			switch {
			case score < pot.Score:
				pot.Score, pot.Cards, pot.Winners = score, cards, []int{seat}
			case score == pot.Score:
				pot.Winners = append(pot.Winners, seat)
			}
		}
		pot.Hand = HandName(pot.Score, pot.Cards)

		share := pot.Amount / uint(len(pot.Winners))
		for _, seat := range pot.Winners {
			s.Player(seat).Stack += share
		}
		odd := s.next(s.Button, func(p *Player) bool { return slices.Contains(pot.Winners, p.Seat) })
		s.Player(odd).Stack += pot.Amount % uint(len(pot.Winners))
		events = append(events, PotWon{Pot: i, Amount: pot.Amount, Winners: pot.Winners, Hand: pot.Hand})
	}
	return append(events, s.endHand()...)
}

// endHand clears the hand, keeping the cards, board and pots up to show the
// result. Busted players sit out.
func (s *State) endHand() []Event {
	s.Street = NoStreet
	s.Betting = false
	s.ToAct = 0
	s.Deck = nil
	for i := range s.Players {
		p := &s.Players[i]
		p.Bet, p.Committed, p.In, p.Acted = 0, 0, false, false
		if p.Stack == 0 {
			p.SittingOut = true
		}
	}
	return []Event{HandEnded{s.Hands}}
}
