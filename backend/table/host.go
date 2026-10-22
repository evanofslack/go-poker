package table

import (
	"fmt"
	"slices"
	"time"

	"github.com/evanofslack/go-poker/bot"
)

func (r *reducer) hostAction(p *Person, c HostAction) error {
	if p.Name != r.s.Host {
		return errNotHost
	}
	switch c.Kind {
	case MakeHost:
		return r.makeHost(p, c.Name)
	case SetBanker:
		if !r.s.connected(c.Name) && !r.s.inLedger(c.Name) {
			return fail("%s isn't at the table", c.Name)
		}
		r.s.Banker = c.Name
		r.log("%s made %s the banker", p.Name, c.Name)
	case MoveToSpectator:
		n := r.s.seatNamed(c.Name)
		if n == 0 {
			return fail("%s doesn't have a seat", c.Name)
		}
		r.log("%s moved %s to watching", p.Name, c.Name)
		r.unseat(n)
	case RemovePlayer:
		return r.removePlayer(p, c.Name)
	case SitOutPlayer:
		return r.sitOutPlayer(p, c.Name)
	case ApproveBuy:
		return r.approveBuy(p, c.Name)
	case DenyBuy:
		if _, ok := r.s.Requests[c.Name]; !ok {
			return fail("%s hasn't asked to buy in", c.Name)
		}
		delete(r.s.Requests, c.Name)
		r.log("%s turned down %s's buy-in", p.Name, c.Name)
	default:
		return fail("unknown host action")
	}
	return nil
}

func (r *reducer) makeHost(p *Person, name string) error {
	if name == p.Name {
		return fail("you're already the host")
	}
	if !r.s.connected(name) {
		return fail("%s isn't at the table", name)
	}
	r.s.Host = name
	r.log("%s made %s the host", p.Name, name)
	return nil
}

// removePlayer gives up name's seat and tells their client to leave the
// table. There are no bans: they can join again.
func (r *reducer) removePlayer(p *Person, name string) error {
	if name == p.Name {
		return fail("you can't remove yourself")
	}
	n := r.s.seatNamed(name)
	target := r.s.personNamed(name)
	if target != nil && target.Name != name {
		target = nil
	}
	if n == 0 && (target == nil || !target.Connected) {
		return fail("%s isn't at the table", name)
	}
	r.log("%s removed %s from the table", p.Name, name)
	if n != 0 {
		r.unseat(n)
	}
	if target != nil && target.Connected {
		r.send(target.ID, Error{Code: CodeRemoved, Message: "the host removed you from the table"})
	}
	return nil
}

// sitOutPlayer sits name out: right away between hands, else once the hand
// ends. Only they can sit back in.
func (r *reducer) sitOutPlayer(p *Person, name string) error {
	n := r.s.seatNamed(name)
	if n == 0 {
		return fail("%s doesn't have a seat", name)
	}
	seat := r.s.Seats[n]
	if r.s.Game.Player(n).SittingOut || seat.SitOutNext {
		return fail("%s is already sitting out", name)
	}
	if err := r.s.Game.SetSittingOut(n, true); err != nil {
		seat.SitOutNext = true
		r.log("%s will sit %s out after this hand", p.Name, name)
		return nil
	}
	r.log("%s sat %s out", p.Name, name)
	return nil
}

// adjustChips adds to name's stack, or takes away, between their hands. The
// ledger counts it as chips in or out.
func (r *reducer) adjustChips(p *Person, c AdjustChips) error {
	if p.Name != r.s.Host {
		return errNotHost
	}
	n := r.s.seatNamed(c.Name)
	if n == 0 {
		return fail("%s doesn't have a seat", c.Name)
	}
	g := &r.s.Game
	gp := g.Player(n)
	if gp.In || gp.Committed > 0 {
		return fail("wait for the hand to end")
	}
	note := ""
	if c.Reason != "" {
		note = fmt.Sprintf(" (%s)", c.Reason)
	}

	if c.Amount > 0 {
		add := uint(c.Amount)
		if gp.Stack+add > r.s.Settings.MaxBuy {
			return fail("can't take %s's stack above %d", c.Name, r.s.Settings.MaxBuy)
		}
		busted := gp.Stack == 0
		g.AddChips(n, add)
		// chips deal a busted player back in, like a rebuy
		if busted {
			g.SetSittingOut(n, false)
		}
		r.record(c.Name, entryAdd, add, p.Name, c.Reason)
		r.log("%s added %d to %s's stack%s", p.Name, add, c.Name, note)
		r.scheduleDeal()
		return nil
	}
	take := uint(-c.Amount)
	if take > gp.Stack {
		return fail("%s only has %d", c.Name, gp.Stack)
	}
	g.RemoveChips(n, take)
	// a player left with nothing sits out until they rebuy
	if gp.Stack == 0 {
		g.SetSittingOut(n, true)
	}
	r.record(c.Name, entryRemove, take, p.Name, c.Reason)
	r.log("%s took %d from %s's stack%s", p.Name, take, c.Name, note)
	return nil
}

// pause stops dealing new hands, or starts again. Mid-hand, the hand plays
// out first.
func (r *reducer) pause(p *Person, pause bool) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case s.Paused && pause:
		return fail("the game is already paused")
	case !s.Paused && !pause:
		return fail("the game isn't paused")
	}
	s.Paused = pause
	switch {
	case pause && s.Game.InHand():
		r.log("%s paused the game after this hand", p.Name)
	case pause:
		r.cancel(TimerDeal)
		r.log("%s paused the game", p.Name)
	default:
		r.log("%s resumed the game", p.Name)
		r.scheduleDeal()
	}
	return nil
}

// endSession ends the session: right away between hands, else once the hand
// ends.
func (r *reducer) endSession(p *Person) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case s.Ending:
		return fail("the session already ends after this hand")
	case s.Game.InHand():
		s.Ending = true
		r.log("%s will end the session after this hand", p.Name)
		return nil
	}
	r.log("%s ended the session", p.Name)
	r.end()
	return nil
}

// addBot has the host sit a bot down in an empty seat.
func (r *reducer) addBot(p *Person, c AddBot) error {
	switch {
	case p.Name != r.s.Host:
		return fail("only the host can add bots")
	case !bot.Valid(c.Level):
		return fail("invalid bot level")
	case r.s.nameTaken(c.Name):
		return fail("name taken")
	}
	if err := r.checkSeat(c.Seat, c.BuyIn); err != nil {
		return err
	}
	r.sit(c.Seat, c.BuyIn, &Seat{Name: c.Name, Bot: c.Level, BotBuyIn: c.BuyIn})
	return nil
}

// removeBot has the host take a bot out of the game. Mid-hand it's folded on
// its turn and removed after the hand, like a leaving player.
func (r *reducer) removeBot(p *Person, n int) error {
	if p.Name != r.s.Host {
		return fail("only the host can remove bots")
	}
	seat := r.s.Seats[n]
	if seat == nil || seat.Bot == "" || seat.Leaving {
		return fail("no bot in that seat")
	}
	r.unseat(n)
	return nil
}

// BuyRequest is a rebuy, or a returning player sitting back down, waiting for
// the host on a table that approves rebuys.
type BuyRequest struct {
	Name   string    `json:"name"`
	Amount uint      `json:"amount"`
	Seat   int       `json:"seat"` // the seat asked for, 0 to rebuy at their seat
	At     time.Time `json:"at"`   // when it was asked, for ordering
}

// needsApproval reports whether name's rebuys wait for the host. The host's
// own don't.
func (r *reducer) needsApproval(name string) bool {
	return r.s.Settings.ApproveRebuys && name != r.s.Host
}

// requestBuy puts req in front of the host, replacing any earlier request
// from the same player.
func (r *reducer) requestBuy(req BuyRequest) {
	req.At = r.now
	r.s.Requests[req.Name] = req
	if req.Seat == 0 {
		r.log("%s asks to rebuy for %d", req.Name, req.Amount)
	} else {
		r.log("%s asks to sit back down with %d", req.Name, req.Amount)
	}
}

// BuyRequests lists the requests waiting for the host, oldest first.
func BuyRequests(s *State) []BuyRequest {
	rs := make([]BuyRequest, 0, len(s.Requests))
	for _, req := range s.Requests {
		rs = append(rs, req)
	}
	slices.SortFunc(rs, func(a, b BuyRequest) int { return a.At.Compare(b.At) })
	return rs
}

// approveBuy carries out name's request, checking it again first. One that
// no longer fits is dropped; a rebuy mid-hand waits for the hand to end.
func (r *reducer) approveBuy(p *Person, name string) error {
	req, ok := r.s.Requests[name]
	if !ok {
		return fail("%s hasn't asked to buy in", name)
	}
	drop := func(why string) error {
		delete(r.s.Requests, name)
		r.tell(p.ID, "%s, so their request was dropped", why)
		return nil
	}

	if req.Seat == 0 {
		n := r.s.seatNamed(name)
		if n == 0 {
			return drop(fmt.Sprintf("%s doesn't have a seat", name))
		}
		if gp := r.s.Game.Player(n); gp.In || gp.Committed > 0 {
			return fail("wait for the hand to end")
		}
		if err := r.checkRebuy(n, req.Amount); err != nil {
			return drop(fmt.Sprintf("%s's rebuy isn't allowed anymore", name))
		}
		delete(r.s.Requests, name)
		r.log("%s approved %s's rebuy", p.Name, name)
		r.applyRebuy(n, req.Amount)
		return nil
	}

	target := r.s.personNamed(name)
	switch {
	case target == nil || target.Name != name || !target.Connected:
		return drop(fmt.Sprintf("%s isn't at the table", name))
	case target.Seat != 0:
		return drop(fmt.Sprintf("%s already has a seat", name))
	}
	if r.checkSeat(req.Seat, req.Amount) != nil {
		return drop(fmt.Sprintf("seat %d isn't free anymore", req.Seat))
	}
	delete(r.s.Requests, name)
	r.log("%s approved %s's buy-in", p.Name, name)
	r.seatPerson(target, req.Seat, req.Amount)
	return nil
}

func (r *reducer) rebuy(p *Person, amount uint) error {
	if p.Seat == 0 {
		return fail("take a seat first")
	}
	if err := r.checkRebuy(p.Seat, amount); err != nil {
		return err
	}
	if r.needsApproval(p.Name) {
		r.requestBuy(BuyRequest{Name: p.Name, Amount: amount})
		return nil
	}
	r.applyRebuy(p.Seat, amount)
	return nil
}

// checkRebuy returns why the player in seat can't rebuy for amount, if they can't.
func (r *reducer) checkRebuy(n int, amount uint) error {
	gp := r.s.Game.Player(n)
	set := r.s.Settings
	switch {
	case gp.In || gp.Committed > 0:
		return fail("can't rebuy during a hand")
	case amount == 0:
		return fail("rebuy amount must be more than 0")
	// topping up can be any amount, but a busted player buys back in
	case gp.Stack == 0 && amount < set.MinBuy:
		return fail("rebuy must be at least %d", set.MinBuy)
	case gp.Stack+amount > set.MaxBuy:
		return fail("rebuy can't take your stack above %d", set.MaxBuy)
	}
	return nil
}

// applyRebuy adds amount to the stack in seat n, after checkRebuy, dealing a
// busted player back in.
func (r *reducer) applyRebuy(n int, amount uint) {
	g := &r.s.Game
	busted := g.Player(n).Stack == 0
	g.AddChips(n, amount)
	if busted {
		g.SetSittingOut(n, false)
	}
	name := r.s.Seats[n].Name
	r.record(name, entryRebuy, amount, name, "")
	r.log("%s rebuys for %d", name, amount)
	r.scheduleDeal()
}
