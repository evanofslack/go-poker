package table

import (
	"slices"

	"github.com/evanofslack/go-poker/holdem"
)

// Side games pay chips straight from one stack to another, outside the pots,
// between hands. Nobody pays more than they have. The ledger needs nothing:
// a player's net already follows their stack.

// Side game kinds. The worst hand bounty is 7-2, or J-6 on a short deck.
const (
	KindSevenDeuce = "7-2"
	KindJackSix    = "J-6"
)

// bountyKind is the worst hand bounty in g: 7-2, or J-6 on a short deck.
func bountyKind(g holdem.Game) string {
	if g.Short {
		return KindJackSix
	}
	return KindSevenDeuce
}

// Payment is chips one player paid another for a side game.
type Payment struct {
	Kind   string `json:"kind"` // one of the side game kinds
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint   `json:"amount"`
}

// Paid is a side game paying out.
type Paid struct {
	Payments []Payment `json:"payments"`
}

func (Paid) EventType() string { return "paid" }

// pay moves amount chips from seat from to seat to, or all from has if
// that's less. A payer left with nothing sits out, like any busted player.
func (r *reducer) pay(kind string, from, to int, amount uint) (Payment, bool) {
	g := &r.s.Game
	payer := g.Player(from)
	if payer == nil || g.Player(to) == nil {
		return Payment{}, false
	}
	amt := min(amount, payer.Stack)
	if amt == 0 || g.RemoveChips(from, amt) != nil || g.AddChips(to, amt) != nil {
		return Payment{}, false
	}
	if g.Player(from).Stack == 0 {
		g.SetSittingOut(from, true)
	}
	return Payment{kind, r.s.Seats[from].Name, r.s.Seats[to].Name, amt}, true
}

// isWorstHand reports whether cards are g's worst hand, of any suits: a seven
// and a deuce, or a jack and a six on a short deck. Omaha has none.
func isWorstHand(g holdem.Game, cards []holdem.Card) bool {
	if len(cards) != 2 || cards[0] == 0 || cards[1] == 0 {
		return false
	}
	hi, lo := "7", "2"
	if g.Short {
		hi, lo = "J", "6"
	}
	a, b := cards[0].String()[:1], cards[1].String()[:1]
	return a == hi && b == lo || a == lo && b == hi
}

// winsSevenDeuce reports whether the player in seat won or split the main
// pot of the hand just ended holding 7-2, on a table playing it. They still
// have to show to be paid.
func winsSevenDeuce(s *State, seat int) bool {
	p := s.Game.Player(seat)
	return s.Settings.SevenDeuce > 0 && p != nil && p.Dealt && !s.Game.InHand() &&
		len(s.Game.Pots) > 0 && slices.Contains(s.Game.Pots[0].Winners, seat) && isWorstHand(s.Game.Config.Game, p.Cards)
}

// sevenDeuce pays the worst hand bounty to seat if it won with 7-2, or J-6
// on a short deck, and showed it:
// from everyone else dealt into h who's still at their seat.
func (r *reducer) sevenDeuce(h *HandRecord, seat int) []Payment {
	s := r.s
	if !winsSevenDeuce(s, seat) || !s.Game.Player(seat).Shown {
		return nil
	}
	var paid []Payment
	var total uint
	for _, hp := range h.Players {
		if hp.Seat == seat || s.Seats[hp.Seat] == nil || s.Seats[hp.Seat].Name != hp.Name {
			continue
		}
		if pm, ok := r.pay(bountyKind(s.Game.Config.Game), hp.Seat, seat, s.Settings.SevenDeuce); ok {
			paid = append(paid, pm)
			total += pm.Amount
		}
	}
	if len(paid) > 0 {
		r.log("%s wins the %s: %d from %s", s.Seats[seat].Name, bountyKind(s.Game.Config.Game), total, plural(len(paid), "player"))
		r.res.Events = append(r.res.Events, Paid{paid})
	}
	return paid
}

// ShowClaims are what the player in seat would collect by showing their
// cards now, like "7-2" or "stand-up".
func ShowClaims(s *State, seat int) []string {
	claims := []string{}
	p := s.Game.Player(seat)
	if p == nil || p.Shown {
		return claims
	}
	if winsSevenDeuce(s, seat) {
		claims = append(claims, bountyKind(s.Game.Config.Game))
	}
	if sitsDown(s, seat, s.Seats[seat].Name) {
		claims = append(claims, KindStandUp)
	}
	return claims
}

// lateShow updates the last hand's record after seat showed between hands:
// their cards, and anything showing won them, which also moves everyone's
// nets.
func (r *reducer) lateShow(seat int) {
	s := r.s
	n := len(s.History)
	if n == 0 || s.History[n-1].Hand != s.Game.Hands {
		return
	}
	h := &s.History[n-1]
	hp := h.player(seat)
	if hp == nil {
		return
	}
	hp.Shown = true
	hp.bestHand(s.Game.Config.Game, s.Game.Player(seat).Cards, s.Game.Board)
	r.latePayments(h, append(r.sevenDeuce(h, seat), r.standUp()...))
	r.res.Events = append(r.res.Events, HandUpdated{*h})
}

// latePayments adds side game payments made after h was recorded to it, and
// to the totals.
func (r *reducer) latePayments(h *HandRecord, paid []Payment) {
	for _, pm := range paid {
		h.Payments = append(h.Payments, pm)
		for i := range h.Players {
			switch h.Players[i].Name {
			case pm.From:
				h.Players[i].Net -= int(pm.Amount)
			case pm.To:
				h.Players[i].Net += int(pm.Amount)
			}
		}
		totalsFor(r.s.Totals, pm.From).Net -= int(pm.Amount)
		totalsFor(r.s.Totals, pm.To).Net += int(pm.Amount)
	}
}

// botsShow shows the cards of bots that would collect something by it.
func (r *reducer) botsShow() {
	for _, n := range sortedSeats(r.s) {
		if r.s.Seats[n].Bot == "" || len(ShowClaims(r.s, n)) == 0 {
			continue
		}
		if events, err := r.s.Game.Show(n); err == nil {
			r.res.Events = append(r.res.Events, events...)
			r.lateShow(n)
		}
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return itoa(uint(n)) + " " + word + "s"
}
