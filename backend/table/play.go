package table

import (
	"errors"
	"fmt"
	"time"

	"github.com/evanofslack/go-poker/bot"
	"github.com/evanofslack/go-poker/holdem"
)

// start starts dealing. Any seated player can, and so can the host without a
// seat, for a game of only bots.
func (r *reducer) start(p *Person) error {
	switch {
	case p.Seat == 0 && r.s.Host != p.Name:
		return fail("take a seat first")
	case r.s.Paused:
		return fail("the game is paused")
	case r.s.Running:
		return fail("the game has already started")
	case r.s.Game.ReadyCount() < 2:
		return fail("need at least two ready players")
	}
	r.s.Running = true
	r.dealHand(false)
	return nil
}

// scheduleDeal deals the next hand after the table's pause, if the game is
// running and between hands. It restarts the pause if one is under way.
func (r *reducer) scheduleDeal() {
	s := r.s
	if s.Running && !s.Paused && !s.Ended() && !s.Game.InHand() {
		r.schedule(TimerDeal, time.Duration(s.Settings.NextHandDelay)*time.Millisecond)
	}
}

// dealHand deals a new hand. waiting logs why it can't, for a hand that was
// due.
func (r *reducer) dealHand(waiting bool) {
	s := r.s
	if !s.Running || s.Paused || s.Ended() || s.Game.InHand() {
		return
	}
	events, err := s.Game.StartHand()
	if err != nil {
		if waiting {
			r.log("waiting for players")
		}
		return
	}
	clear(s.Acts)
	r.played(events, false)
}

func (r *reducer) play(p *Person, c Play) error {
	if p.Seat == 0 {
		return fail("take a seat first")
	}
	events, err := r.s.Game.Act(p.Seat, c.Action)
	if errors.Is(err, holdem.ErrNotYourTurn) {
		return fail("not your turn")
	}
	if err != nil {
		return fail("you can't %s now", actionName(c.Action))
	}
	r.played(events, false)
	return nil
}

func actionName(a holdem.Action) string {
	switch a.(type) {
	case holdem.Check:
		return "check"
	case holdem.Call:
		return "call"
	case holdem.Raise:
		return "bet that"
	}
	return "fold"
}

// played takes in what the game did: it records the events, folds players
// who left when it's their turn, and sets the timers for what comes next.
// timeout marks moves the turn timer made.
func (r *reducer) played(events []holdem.Event, timeout bool) {
	for _, e := range events {
		r.res.Events = append(r.res.Events, e)
		r.recordEvent(e, timeout)
	}
	if r.foldLeavers() {
		return
	}

	s := r.s
	r.cancel(TimerTurn, TimerBot, TimerStreet)
	switch {
	case s.Game.Betting:
		if s.Settings.TurnTime > 0 {
			r.schedule(TimerTurn, time.Duration(s.Settings.TurnTime)*time.Millisecond)
		}
		if s.Seats[s.Game.ToAct].Bot != "" {
			r.schedule(TimerBot, s.Timing.BotDelay+time.Duration(s.rand().Int64N(int64(s.Timing.BotDelay)+1)))
		}
	case s.Game.InHand():
		// show the street's last action before dealing the next
		r.schedule(TimerStreet, s.Timing.StreetDelay)
	default:
		r.afterHand()
	}
}

// foldLeavers folds the player to act if they left, and reports whether it
// did, having dealt with what follows.
func (r *reducer) foldLeavers() bool {
	g := &r.s.Game
	if !g.Betting || !r.s.Seats[g.ToAct].Leaving {
		return false
	}
	events, err := g.Act(g.ToAct, holdem.Fold{})
	if err != nil {
		return false
	}
	r.played(events, false)
	return true
}

func (r *reducer) nextStreet() {
	events, err := r.s.Game.NextStreet()
	if err != nil {
		return
	}
	r.played(events, false)
}

// turnTimedOut checks for the player to act if they can, and folds otherwise.
func (r *reducer) turnTimedOut() {
	g := &r.s.Game
	var a holdem.Action = holdem.Fold{}
	if g.Legal(g.ToAct).Check {
		a = holdem.Check{}
	}
	if events, err := g.Act(g.ToAct, a); err == nil {
		r.played(events, true)
	}
}

// botMove makes the move of the bot to act, from what its player can see.
func (r *reducer) botMove() {
	g := &r.s.Game
	seat := r.s.Seats[g.ToAct]
	if !g.Betting || seat == nil || seat.Bot == "" {
		return
	}
	a := bot.Decide(seat.Bot, g.View(g.ToAct), g.ToAct, r.s.rand())
	events, err := g.Act(g.ToAct, a)
	if err != nil {
		// a bad move from the bot shouldn't stall the table
		events, err = g.Act(g.ToAct, holdem.Check{})
		if err != nil {
			events, err = g.Act(g.ToAct, holdem.Fold{})
		}
	}
	if err == nil {
		r.played(events, false)
	}
}

// afterHand tidies up once a hand ends: players who left go, sit-outs take
// effect, then the session ends or the next hand is scheduled.
func (r *reducer) afterHand() {
	r.removeLeavers()
	r.applySitOuts()
	if r.s.Ending {
		r.end()
		return
	}
	r.rebuyBots()
	r.scheduleDeal()
}

// rebuyBots buys busted bots back in for their buy-in.
func (r *reducer) rebuyBots() {
	for _, n := range sortedSeats(r.s) {
		seat := r.s.Seats[n]
		if seat.Bot == "" || seat.Leaving || r.s.Game.Player(n).Stack > 0 {
			continue
		}
		if r.s.Game.AddChips(n, seat.BotBuyIn) != nil || r.s.Game.SetSittingOut(n, false) != nil {
			continue
		}
		r.record(seat.Name, entryRebuy, seat.BotBuyIn, seat.Name, "")
		r.log("%s rebuys for %d", seat.Name, seat.BotBuyIn)
	}
}

// end stops dealing for good, so the ledger is final, and closes the table
// after Timing.EndedTTL. Only between hands.
func (r *reducer) end() {
	s := r.s
	s.Ending = false
	s.EndedAt = r.now
	clear(s.Requests)
	r.cancel(TimerDeal, TimerTurn, TimerBot, TimerIdle)
	r.schedule(TimerClose, s.Timing.EndedTTL)
	r.log("the session has ended, the ledger is final")
}

// log adds a line to the table's log.
func (r *reducer) log(format string, args ...any) {
	s := r.s
	s.LogSeq++
	l := Log{ID: s.LogSeq, Message: fmt.Sprintf(format, args...), Time: r.now.UnixMilli()}
	s.Logs = append(s.Logs, l)
	if len(s.Logs) > maxLogs {
		s.Logs = s.Logs[len(s.Logs)-maxLogs:]
	}
	r.res.Events = append(r.res.Events, Logged{l})
}

// maxLogs caps how many log lines a table keeps for people who join later.
const maxLogs = 200
