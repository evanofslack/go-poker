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

// dealHand deals a new hand, once nobody is being asked to straddle.
// waiting logs why it can't, for a hand that was due.
func (r *reducer) dealHand(waiting bool) {
	s := r.s
	if !s.Running || s.Paused || s.Ended() || s.Game.InHand() {
		return
	}
	r.askStraddles()
	if s.Game.InHand() {
		return // the asking ended and dealt
	}
	straddles := 0
	if a := s.StraddleAsk; a != nil {
		if a.Seat != 0 {
			a.Waiting = true
			return
		}
		straddles = len(a.Yes)
	}
	deal := holdem.Deal{Straddles: straddles}
	if s.SpecialNext != "" {
		deal = holdem.Deal{Bomb: s.Settings.BombPot}
		if isDouble(s.SpecialNext) {
			deal.Boards = 2
		}
	}
	events, err := s.Game.StartHand(deal)
	if err != nil {
		if waiting {
			r.log("waiting for players")
		}
		return
	}
	r.cancel(TimerDeal)
	r.dropStraddles()
	s.SpecialNext = ""
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
		if r.askRuns() {
			break
		}
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
	if r.autoTime(g.ToAct) {
		return
	}
	var a holdem.Action = holdem.Fold{}
	if g.Legal(g.ToAct).Check {
		a = holdem.Check{}
	}
	if events, err := g.Act(g.ToAct, a); err == nil {
		r.played(events, true)
	}
}

// show turns up the person's cards from the last hand.
func (r *reducer) show(p *Person) error {
	if p.Seat == 0 {
		return fail("take a seat first")
	}
	events, err := r.s.Game.Show(p.Seat)
	if err != nil {
		return fail("you have no cards to show")
	}
	r.res.Events = append(r.res.Events, events...)
	r.lateShow(p.Seat)
	return nil
}

// huntShow is how long hunted cards stay up at least, before the next hand.
const huntShow = 3 * time.Second

// hunt turns up the rest of the board after a hand that ended early, for a
// player who was dealt in, on tables that allow it.
func (r *reducer) hunt(p *Person) error {
	s := r.s
	if !s.Settings.DuckHunt {
		return fail("duck hunting isn't allowed at this table")
	}
	if gp := s.Game.Player(p.Seat); p.Seat == 0 || gp == nil || !gp.Dealt {
		return fail("only players dealt into the last hand can duck hunt")
	}
	events, err := s.Game.Hunt()
	if err != nil {
		return fail("there's nothing to hunt")
	}
	r.res.Events = append(r.res.Events, events...)
	if at, ok := s.Timers[TimerDeal]; ok && at.Sub(r.now) < huntShow {
		r.schedule(TimerDeal, huntShow)
	}
	return nil
}

// botMove makes the move of the bot to act, from what its player can see.
func (r *reducer) botMove() {
	g := &r.s.Game
	seat := r.s.Seats[g.ToAct]
	if !g.Betting || seat == nil || seat.Bot == "" {
		return
	}
	reads := map[int]bot.Read{}
	for _, p := range g.Players {
		if s := r.s.Seats[p.Seat]; s != nil && p.Seat != g.ToAct {
			if t := r.s.Totals[s.Name]; t != nil {
				reads[p.Seat] = bot.Read{Hands: t.Hands, VPIP: t.VPIP, PFR: t.PFR, ThreeBetSpots: t.ThreeBetSpots, ThreeBets: t.ThreeBets, Aggressive: t.Aggressive, Calls: t.Calls}
			}
		}
	}
	a := bot.Decide(seat.Bot, seat.Name, g.View(g.ToAct), g.ToAct, r.s.Current.line(), reads, r.s.rand())
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
	r.botsShow()
	r.earnTime()
	r.removeLeavers()
	r.applySitOuts()
	if r.s.Ending {
		r.end()
		return
	}
	if r.s.Next != nil {
		r.applySettings("settings changed:", *r.s.Next)
	}
	r.rebuyBots()
	r.scheduleSpecial()
	r.scheduleDeal()
}

// rebuyBots buys busted bots back in for their buy-in.
func (r *reducer) rebuyBots() {
	for _, n := range sortedSeats(r.s) {
		seat := r.s.Seats[n]
		if seat.Bot == "" || seat.Leaving || r.s.Game.Player(n).Stack > 0 {
			continue
		}
		// the buy-in limits may have changed since the bot sat down
		buyIn := min(max(seat.BotBuyIn, r.s.Settings.MinBuy), r.s.Settings.MaxBuy)
		if r.s.Game.AddChips(n, buyIn) != nil || r.s.Game.SetSittingOut(n, false) != nil {
			continue
		}
		r.record(seat.Name, entryRebuy, buyIn, seat.Name, "")
		r.log("%s rebuys for %d", seat.Name, buyIn)
	}
}

// end stops dealing for good, so the ledger is final, and closes the table
// after Timing.EndedTTL. Only between hands.
func (r *reducer) end() {
	s := r.s
	s.Ending = false
	s.EndedAt = r.now
	clear(s.Requests)
	s.Next = nil
	s.StandUp, s.LastStandUp = nil, 0
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
	if len(s.Logs) > MaxLogs {
		s.Logs = s.Logs[len(s.Logs)-MaxLogs:]
	}
	r.res.Events = append(r.res.Events, Logged{l})
}

// MaxLogs caps how many log lines a table keeps for people who join later.
const MaxLogs = 200
