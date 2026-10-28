package table

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func (r *reducer) join(j Join) error {
	s := r.s
	if j.Token != "" {
		hash := hashToken(j.Token)
		for _, seat := range s.Seats {
			if p := s.People[seat.Person]; p != nil && seat.Token == hash && !seat.Leaving {
				p.Connected = true
				r.cancel(graceTimer(p.ID))
				r.res.Grace = "reconnected"
				r.log("%s reconnected", p.Name)
				r.res.Joined = &Joined{ID: p.ID, Name: p.Name, Reclaimed: true}
				return nil
			}
		}
	}
	switch {
	case s.Password != "" && j.Password == "":
		return ErrNeedPassword
	case s.Password != "" && !checkPassword(s.Password, j.Password):
		return ErrWrongPassword
	case s.nameTaken(j.Name):
		return ErrNameTaken
	}

	s.People[j.ID] = &Person{ID: j.ID, Name: j.Name, Connected: true}
	if s.Host == "" {
		s.Host = j.Name
	}
	r.res.Joined = &Joined{ID: j.ID, Name: j.Name}
	msg := fmt.Sprintf("%s has joined", j.Name)
	if s.inLedger(j.Name) {
		msg = fmt.Sprintf("%s rejoined, continuing their ledger", j.Name)
	}
	r.send("", ChatMessage{SystemName, msg, r.now.UnixMilli()})
	return nil
}

// leave takes p away from the table. A seated person who disconnected keeps
// their seat, and the host role, for the reconnect grace period.
func (r *reducer) leave(p *Person, disconnected bool) {
	s := r.s
	if disconnected && p.Seat != 0 {
		p.Connected = false
		r.log("%s disconnected", p.Name)
		r.schedule(graceTimer(p.ID), s.Timing.ReconnectGrace)
		return
	}
	if p.Seat != 0 {
		r.unseat(p.Seat)
	}
	delete(s.People, p.ID)
	delete(s.Requests, p.Name)
	if s.Host == p.Name {
		r.passHost()
	}
	r.closeIfEmpty()
}

func graceID(t Timer) (string, bool) {
	return strings.CutPrefix(string(t), "grace:")
}

// graceExpired removes a disconnected person who didn't come back in time.
func (r *reducer) graceExpired(id string) {
	p := r.s.People[id]
	if p == nil {
		return
	}
	r.res.Grace = "expired"
	if p.Seat != 0 {
		r.unseat(p.Seat)
	}
	delete(r.s.People, id)
	if r.s.Host == p.Name {
		r.passHost()
	}
	r.closeIfEmpty()
}

// closeIfEmpty closes the table once nobody is connected and nobody is
// waiting to reconnect. An ended table stays readable until it times out.
func (r *reducer) closeIfEmpty() {
	if len(r.s.People) == 0 && !r.s.Ended() {
		r.res.Close = &Close{"empty", "the table is empty"}
	}
}

// passHost gives the host role to a connected seated person, else to the
// connected person first by name, else nobody.
func (r *reducer) passHost() {
	s := r.s
	s.Host = ""
	for _, p := range s.Game.Players {
		seat := s.Seats[p.Seat]
		if person := s.People[seat.Person]; person != nil && person.Connected {
			s.Host = person.Name
			return
		}
	}
	for _, p := range s.People {
		if p.Connected && (s.Host == "" || p.Name < s.Host) {
			s.Host = p.Name
		}
	}
}

// checkSeat returns why a new player can't sit in seat with buyIn, if they can't.
func (r *reducer) checkSeat(seat int, buyIn uint) error {
	set := r.s.Settings
	if seat < 1 || seat > int(set.Seats) {
		return fail("invalid seat")
	}
	if buyIn < set.MinBuy || buyIn > set.MaxBuy {
		return fail("buy-in must be between %d and %d", set.MinBuy, set.MaxBuy)
	}
	if r.s.Seats[seat] != nil {
		return fail("seat taken")
	}
	return nil
}

func (r *reducer) takeSeat(p *Person, seat int, buyIn uint) error {
	if p.Seat != 0 {
		return fail("you already have a seat")
	}
	if err := r.checkSeat(seat, buyIn); err != nil {
		return err
	}
	// sitting back down is a rebuy, so leaving can't skip approval
	if r.needsApproval(p.Name) && r.s.inLedger(p.Name) {
		r.requestBuy(BuyRequest{Name: p.Name, Amount: buyIn, Seat: seat})
		return nil
	}
	r.seatPerson(p, seat, buyIn)
	return nil
}

// seatPerson sits p down, after checkSeat, and gives them the seat's token.
func (r *reducer) seatPerson(p *Person, seat int, buyIn uint) {
	token, hash := newToken()
	r.sit(seat, buyIn, &Seat{Name: p.Name, Person: p.ID, Token: hash, TimeCards: r.s.Settings.TimeBank})
	p.Seat = seat
	r.send(p.ID, NewToken{token})
}

// sit puts a player in seat, after checkSeat.
func (r *reducer) sit(n int, buyIn uint, seat *Seat) {
	if err := r.s.Game.Sit(n, buyIn); err != nil {
		panic(err) // checkSeat said it was free
	}
	r.s.Seats[n] = seat
	r.record(seat.Name, entryBuyIn, buyIn, seat.Name, "")
	r.log("%s sits down with %d", seat.Name, buyIn)
	r.scheduleDeal()
}

// unseat gives up seat. Their person stays at the table watching. Mid-hand
// they're folded on their turn and removed after the hand.
func (r *reducer) unseat(n int) {
	seat := r.s.Seats[n]
	if seat == nil || seat.Leaving {
		return
	}
	if p := r.s.People[seat.Person]; p != nil {
		p.Seat = 0
		r.cancel(graceTimer(p.ID))
		r.send(p.ID, NewToken{})
		// a disconnected person without a seat has nothing to wait for
		if !p.Connected {
			delete(r.s.People, p.ID)
		}
	}
	seat.Person = ""
	seat.Token = ""
	seat.Leaving = true
	if r.s.Game.InHand() {
		r.foldLeavers()
		return
	}
	r.removeLeavers()
}

// removeLeavers takes out of the game the players who left, between hands,
// cashing them out.
func (r *reducer) removeLeavers() {
	for _, n := range sortedSeats(r.s) {
		seat := r.s.Seats[n]
		if !seat.Leaving {
			continue
		}
		if r.s.Game.InHand() {
			return // mid-hand, after it then
		}
		r.leaveStandUp(seat.Name)
		stack := r.s.Game.Player(n).Stack
		if err := r.s.Game.Leave(n); err != nil {
			return
		}
		delete(r.s.Seats, n)
		delete(r.s.Acts, n)
		r.record(seat.Name, entryCashOut, stack, seat.Name, "")
		r.log("%s left the table", seat.Name)
	}
}

// sortedSeats are the taken seat numbers in order.
func sortedSeats(s *State) []int {
	var ns []int
	for n := range s.Seats {
		ns = append(ns, n)
	}
	slices.Sort(ns)
	return ns
}

func (r *reducer) sitOut(p *Person) error {
	if p.Seat == 0 {
		return fail("take a seat first")
	}
	seat := r.s.Seats[p.Seat]
	gp := r.s.Game.Player(p.Seat)
	switch {
	case seat.SitOutNext:
		seat.SitOutNext = false
		r.log("%s will play the next hand", p.Name)
	case !gp.SittingOut:
		if err := r.s.Game.SetSittingOut(p.Seat, true); err != nil {
			// sitting out mid-hand could move the button, so wait for it to end
			seat.SitOutNext = true
			r.log("%s will sit out the next hand", p.Name)
			return nil
		}
		r.log("%s sits out", p.Name)
	case gp.Stack == 0:
		return fail("you're out of chips, rebuy first")
	default:
		if err := r.s.Game.SetSittingOut(p.Seat, false); err != nil {
			return fail("can't sit back in right now")
		}
		r.log("%s is back", p.Name)
		r.scheduleDeal()
	}
	return nil
}

// applySitOuts sits out the players who asked to during the last hand.
func (r *reducer) applySitOuts() {
	for _, n := range sortedSeats(r.s) {
		seat := r.s.Seats[n]
		if !seat.SitOutNext {
			continue
		}
		seat.SitOutNext = false
		if r.s.Game.Player(n).SittingOut {
			continue // busted players already are
		}
		if err := r.s.Game.SetSittingOut(n, true); err == nil {
			r.log("%s sits out", seat.Name)
		}
	}
}

var errNotHost = errors.New("only the host can do that")
