package holdem

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"testing"
)

// chips is every chip in the game: stacks plus what's been put in this hand.
func chips(s *State) uint {
	var n uint
	for _, p := range s.Players {
		n += p.Stack + p.Committed
	}
	return n
}

// check fails t if s breaks an invariant.
func check(t *testing.T, s *State, want uint) {
	t.Helper()
	if got := chips(s); got != want {
		t.Fatalf("chips %d, want %d", got, want)
	}
	var committed, potted uint
	for _, p := range s.Players {
		committed += p.Committed
		if p.Bet > p.Committed {
			t.Fatalf("seat %d bet %d more than committed %d", p.Seat, p.Bet, p.Committed)
		}
	}
	for _, pot := range s.Pots {
		potted += pot.Amount
	}
	if s.InHand() && committed != potted {
		t.Fatalf("pots hold %d, players put in %d", potted, committed)
	}
	if s.Betting {
		p := s.Player(s.ToAct)
		if p == nil || !p.canBet() || !s.Legal(s.ToAct).Turn {
			t.Fatalf("seat %d to act can't: %+v", s.ToAct, p)
		}
	} else if s.ToAct != 0 {
		t.Fatalf("seat %d to act while not betting", s.ToAct)
	}
	if !s.InHand() && (s.Betting || len(s.Deck) > 0) {
		t.Fatal("between hands with betting or a deck")
	}
}

// randomAction is a legal move for l, or now and then an illegal one.
func randomAction(r *rand.Rand, l Legal) Action {
	switch r.IntN(10) {
	case 0:
		return Fold{}
	case 1:
		return Raise{To: r.UintN(l.RaiseMax + 50)} // often illegal
	case 2, 3:
		if l.RaiseMax > 0 {
			if r.IntN(4) == 0 {
				return Raise{To: l.RaiseMax}
			}
			return Raise{To: l.RaiseMin + r.UintN(l.RaiseMax-l.RaiseMin+1)}
		}
	}
	if l.Check {
		return Check{}
	}
	return Call{}
}

// tryState runs f on a copy of s; if f fails, s must not have changed.
func try(t *testing.T, s *State, f func() error) {
	t.Helper()
	before := s.Clone()
	if err := f(); err != nil && !reflect.DeepEqual(before, *s) {
		t.Fatalf("failed action changed the state: %v", err)
	}
}

func TestRollout(t *testing.T) {
	for game := range 300 {
		r := rand.New(rand.NewPCG(uint64(game), 1))
		s := New(Config{SmallBlind: 1 + r.UintN(10), BigBlind: 20}, [32]byte{byte(game), byte(game >> 8)})
		seats := 2 + r.IntN(8)
		for seat := 1; seat <= seats; seat++ {
			if r.IntN(4) > 0 {
				s.Sit(seat, 1+r.UintN(2000))
			}
		}
		total := chips(&s)

		for range 40 {
			// between hands: people come and go
			for _, p := range s.Players {
				switch r.IntN(12) {
				case 0:
					try(t, &s, func() error { return s.SetSittingOut(p.Seat, !p.SittingOut) })
				case 1:
					n := 1 + r.UintN(500)
					try(t, &s, func() error { return s.AddChips(p.Seat, n) })
					if s.Player(p.Seat).Stack == p.Stack+n {
						total += n
					}
				case 2:
					if p.Stack > 0 {
						n := 1 + r.UintN(p.Stack)
						try(t, &s, func() error { return s.RemoveChips(p.Seat, n) })
						total -= n
					}
				}
			}
			if _, err := s.StartHand(); err != nil {
				// sit everyone with chips back in and top up
				for _, p := range s.Players {
					if p.Stack == 0 {
						s.AddChips(p.Seat, 100)
						total += 100
					}
					s.SetSittingOut(p.Seat, false)
				}
				if s.ReadyCount() < 2 {
					break
				}
				continue
			}
			check(t, &s, total)

			for steps := 0; s.InHand(); steps++ {
				if steps > 200 {
					t.Fatalf("game %d: hand didn't end", game)
				}
				if !s.Betting {
					if _, err := s.NextStreet(); err != nil {
						t.Fatalf("game %d: next street: %v", game, err)
					}
					check(t, &s, total)
					continue
				}
				// someone acts out of turn now and then
				if r.IntN(20) == 0 {
					other := s.Players[r.IntN(len(s.Players))].Seat
					if other != s.ToAct {
						try(t, &s, func() error { _, err := s.Act(other, Call{}); return err })
					}
				}
				a := randomAction(r, s.Legal(s.ToAct))
				try(t, &s, func() error { _, err := s.Act(s.ToAct, a); return err })
				check(t, &s, total)
				// mid-hand seat changes are refused
				if p := s.Player(s.ToAct); p != nil {
					try(t, &s, func() error { return s.Leave(p.Seat) })
				}
			}

			// the state survives a round trip through JSON
			b, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			var back State
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(normalize(back), normalize(s)) {
				t.Fatalf("game %d: JSON round trip changed the state", game)
			}
		}
	}
}

// normalize makes empty slices nil, which JSON doesn't keep apart.
func normalize(s State) State {
	s = s.Clone()
	if len(s.Board) == 0 {
		s.Board = nil
	}
	if len(s.Deck) == 0 {
		s.Deck = nil
	}
	if len(s.Pots) == 0 {
		s.Pots = nil
	}
	for i := range s.Pots {
		p := &s.Pots[i]
		if len(p.Winners) == 0 {
			p.Winners = nil
		}
		if len(p.Cards) == 0 {
			p.Cards = nil
		}
		if len(p.Eligible) == 0 {
			p.Eligible = nil
		}
	}
	return s
}
