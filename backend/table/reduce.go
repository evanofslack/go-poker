package table

import (
	"errors"
	"fmt"
	"time"
)

// Result is what a command did, for the runtime to carry out.
type Result struct {
	Events  []Event // for everyone, as EventsFor filters them
	Out     []Out
	Joined  *Joined // a Join succeeded
	Close   *Close  // the table should close
	Changed bool    // views need sending again
	Grace   string  // "reconnected" or "expired" when a disconnected player did either

	// Checkpoint is set when no hand is being played, so the table is worth
	// saving: a table brought back from it replays nothing half done.
	Checkpoint bool
}

// Out is a message for one person, or for everyone when To is empty: an
// Error, ChatMessage, NewToken or HandsPage.
type Out struct {
	To  string
	Msg any
}

// Joined is who a Join joined as.
type Joined struct {
	ID        string
	Name      string
	Reclaimed bool // took back a seat with a token: close the person's other connections
}

// Close is the table closing, with Reason for metrics and Message for people.
type Close struct {
	Reason  string
	Message string
}

var (
	ErrNeedPassword  = &Error{"password-required", "this table needs a password"}
	ErrWrongPassword = &Error{"wrong-password", "wrong password"}
	ErrNameTaken     = &Error{"", "name already taken at this table"}
	ErrNoPerson      = &Error{"", "join the table first"}
)

// codes for errors that tell a client to leave the table
const (
	CodeRemoved = "removed"
	CodeClosed  = "closed"
)

// CodeRestarting is the server going down for a restart. The table is saved,
// so the client keeps its seat and rejoins once it reconnects.
const CodeRestarting = "restarting"

func fail(format string, args ...any) error {
	return &Error{Message: fmt.Sprintf(format, args...)}
}

type reducer struct {
	s     *State
	now   time.Time
	res   Result
	quiet bool // nothing in the views changed
}

// Reduce applies cmd to s at now.
func Reduce(s *State, now time.Time, cmd Command) Result {
	r := &reducer{s: s, now: now}
	if err := r.do(cmd); err != nil {
		var e *Error
		if !errors.As(err, &e) {
			e = &Error{Message: err.Error()}
		}
		if j, ok := cmd.(Join); ok {
			r.send(j.ID, *e)
		} else {
			r.send(personOf(cmd), *e)
		}
		return r.res
	}
	r.askStraddles()
	r.res.Changed = !r.quiet
	r.res.Checkpoint = !s.Game.InHand()
	return r.res
}

// personOf is who sent cmd, empty for a timer.
func personOf(cmd Command) string {
	switch c := cmd.(type) {
	case Leave:
		return c.Person
	case Disconnect:
		return c.Person
	case Chat:
		return c.Person
	case TakeSeat:
		return c.Person
	case Start:
		return c.Person
	case Play:
		return c.Person
	case LeaveSeat:
		return c.Person
	case SitOut:
		return c.Person
	case Rebuy:
		return c.Person
	case Show:
		return c.Person
	case Hunt:
		return c.Person
	case AddTime:
		return c.Person
	case GetHands:
		return c.Person
	case Straddle:
		return c.Person
	case StraddlePreset:
		return c.Person
	case RunIt:
		return c.Person
	case AddBot:
		return c.Person
	case RemoveBot:
		return c.Person
	case HostAction:
		return c.Person
	case AdjustChips:
		return c.Person
	case Pause:
		return c.Person
	case EndSession:
		return c.Person
	case StartStandUp:
		return c.Person
	case SpecialHand:
		return c.Person
	case CancelStandUp:
		return c.Person
	case UpdateSettings:
		return c.Person
	case SetPassword:
		return c.Person
	}
	return ""
}

// closedWhenEnded reports whether cmd would deal, seat or move chips, which
// stops once the session ends so the ledger stays final.
func closedWhenEnded(cmd Command) bool {
	switch c := cmd.(type) {
	case TakeSeat, Start, Rebuy, SitOut, Straddle, AdjustChips, Pause, EndSession, AddBot, RemoveBot, StartStandUp, CancelStandUp,
		UpdateSettings, SetPassword:
		return true
	case HostAction:
		return c.Kind == SitOutPlayer || c.Kind == ApproveBuy || c.Kind == DenyBuy
	}
	return false
}

func (r *reducer) do(cmd Command) error {
	if t, ok := cmd.(TimerFired); ok {
		return r.timer(t)
	}
	if j, ok := cmd.(Join); ok {
		r.active()
		return r.join(j)
	}

	p := r.s.People[personOf(cmd)]
	if _, leaving := cmd.(Leave); leaving && p == nil {
		r.quiet = true // already gone, like after the host removed them
		return nil
	}
	if p == nil {
		return ErrNoPerson
	}
	if _, ok := cmd.(Disconnect); !ok {
		r.active()
	}
	if closedWhenEnded(cmd) && r.s.Ended() {
		return fail("the session has ended")
	}

	switch c := cmd.(type) {
	case Leave:
		r.leave(p, false)
	case Disconnect:
		r.leave(p, true)
	case Chat:
		r.quiet = true
		r.send("", ChatMessage{p.Name, c.Text, r.now.UnixMilli()})
	case TakeSeat:
		return r.takeSeat(p, c.Seat, c.BuyIn)
	case Start:
		return r.start(p)
	case Play:
		return r.play(p, c)
	case LeaveSeat:
		if p.Seat == 0 {
			return fail("you don't have a seat")
		}
		r.unseat(p.Seat)
	case SitOut:
		return r.sitOut(p)
	case Rebuy:
		return r.rebuy(p, c.Amount)
	case Show:
		return r.show(p)
	case Hunt:
		return r.hunt(p)
	case AddTime:
		return r.addTime(p)
	case Straddle:
		return r.straddle(p, c.Yes)
	case StraddlePreset:
		return r.straddlePreset(p, c.Preset)
	case RunIt:
		return r.runIt(p, c.Runs)
	case GetHands:
		r.quiet = true
		hands, more := HistoryFor(r.s, p.Name, c.Before)
		r.send(p.ID, HandsPage{hands, more})
	case AddBot:
		return r.addBot(p, c)
	case RemoveBot:
		return r.removeBot(p, c.Seat)
	case HostAction:
		return r.hostAction(p, c)
	case AdjustChips:
		return r.adjustChips(p, c)
	case Pause:
		return r.pause(p, c.Pause)
	case EndSession:
		return r.endSession(p)
	case StartStandUp:
		return r.startStandUp(p, c.Payout)
	case SpecialHand:
		return r.specialHand(p, c.Kind)
	case CancelStandUp:
		return r.cancelStandUp(p)
	case UpdateSettings:
		return r.updateSettings(p, c.Settings)
	case SetPassword:
		return r.setPassword(p, c.Password)
	default:
		return fail("unknown command")
	}
	return nil
}

// active pushes back the idle timeout after a message from a person.
func (r *reducer) active() {
	if !r.s.Ended() {
		r.s.Timers[TimerIdle] = r.now.Add(r.s.Timing.IdleTimeout)
	}
}

// send queues msg for the person with id, or everyone when id is empty.
func (r *reducer) send(id string, msg any) {
	r.res.Out = append(r.res.Out, Out{id, msg})
}

// tell sends person id an error without refusing the command.
func (r *reducer) tell(id, format string, args ...any) {
	r.send(id, Error{Message: fmt.Sprintf(format, args...)})
}

// schedule arms timer for d from now.
func (r *reducer) schedule(timer Timer, d time.Duration) {
	r.s.Timers[timer] = r.now.Add(d)
}

func (r *reducer) cancel(timers ...Timer) {
	for _, t := range timers {
		delete(r.s.Timers, t)
	}
}

func (r *reducer) timer(t TimerFired) error {
	at, ok := r.s.Timers[t.Timer]
	if !ok || !at.Equal(t.At) {
		r.quiet = true // cancelled or moved
		return nil
	}
	delete(r.s.Timers, t.Timer)

	switch t.Timer {
	case TimerDeal:
		r.dealHand(true)
	case TimerStreet:
		r.nextStreet()
	case TimerTurn:
		r.turnTimedOut()
	case TimerBot:
		r.botMove()
	case TimerStraddle:
		r.endStraddles()
	case TimerRunVote:
		if r.s.RunVote != nil {
			r.endRunVote()
		}
	case TimerClose:
		r.res.Close = &Close{"ended", "the table has closed"}
	case TimerIdle:
		r.res.Close = &Close{"idle", fmt.Sprintf("the table closed after %s idle", humanDuration(r.s.Timing.IdleTimeout))}
	default:
		if id, ok := graceID(t.Timer); ok {
			r.graceExpired(id)
		}
	}
	return nil
}

// humanDuration spells out whole hours or minutes, e.g. "30 minutes".
func humanDuration(d time.Duration) string {
	for _, u := range []struct {
		d    time.Duration
		name string
	}{{time.Hour, "hour"}, {time.Minute, "minute"}} {
		if n := d / u.d; n > 0 && d%u.d == 0 {
			if n == 1 {
				return "1 " + u.name
			}
			return fmt.Sprintf("%d %ss", n, u.name)
		}
	}
	return d.String()
}
