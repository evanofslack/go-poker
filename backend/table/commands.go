package table

import (
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// Command is something done to a table: by a person, by the host, or by a
// timer coming due.
type Command interface {
	command()
}

// Join adds a person. The runtime picks ID; a Token that reclaims a seat
// rejoins as that seat's person instead, without the password.
type Join struct {
	ID       string
	Name     string
	Token    string
	Password string
}

// Leave is a person leaving the table on purpose.
type Leave struct{ Person string }

// Disconnect is a person's connection dropping. Seated people keep their
// seat for Timing.ReconnectGrace.
type Disconnect struct{ Person string }

type Chat struct {
	Person string
	Text   string
}

type TakeSeat struct {
	Person string
	Seat   int
	BuyIn  uint
}

// Start starts dealing hands.
type Start struct{ Person string }

// Play is a move on the person's turn.
type Play struct {
	Person string
	Action holdem.Action
}

type LeaveSeat struct{ Person string }

// SitOut toggles sitting out: right away between hands, else after the hand.
type SitOut struct{ Person string }

type Rebuy struct {
	Person string
	Amount uint
}

// Show turns up the person's cards from the last hand, between hands.
type Show struct{ Person string }

// Hunt turns up the rest of the board after a hand that ended early.
type Hunt struct{ Person string }

// GetHands asks for the hands played before hand number Before.
type GetHands struct {
	Person string
	Before int
}

// host commands

type AddBot struct {
	Person string
	Seat   int
	BuyIn  uint
	Name   string
	Level  string
}

type RemoveBot struct {
	Person string
	Seat   int
}

// HostAction is a host action that names a player.
type HostAction struct {
	Person string
	Kind   HostKind
	Name   string
}

type HostKind string

const (
	MakeHost        HostKind = "make-host"
	SetBanker       HostKind = "set-banker"
	MoveToSpectator HostKind = "move-to-spectator"
	RemovePlayer    HostKind = "remove-player"
	SitOutPlayer    HostKind = "sit-out-player"
	ApproveBuy      HostKind = "approve-buy-in"
	DenyBuy         HostKind = "deny-buy-in"
)

// AdjustChips adds Amount to a player's stack, or takes it away when negative.
type AdjustChips struct {
	Person string
	Name   string
	Amount int64
	Reason string
}

type Pause struct {
	Person string
	Pause  bool // false resumes
}

type EndSession struct{ Person string }

// TimerFired is a timer from State.Timers coming due. At is the deadline it
// was armed for, so one that has since moved or gone is ignored.
type TimerFired struct {
	Timer Timer
	At    time.Time
}

func (Join) command()        {}
func (Leave) command()       {}
func (Disconnect) command()  {}
func (Chat) command()        {}
func (TakeSeat) command()    {}
func (Start) command()       {}
func (Play) command()        {}
func (LeaveSeat) command()   {}
func (SitOut) command()      {}
func (Rebuy) command()       {}
func (Show) command()        {}
func (Hunt) command()        {}
func (GetHands) command()    {}
func (AddBot) command()      {}
func (RemoveBot) command()   {}
func (HostAction) command()  {}
func (AdjustChips) command() {}
func (Pause) command()       {}
func (EndSession) command()  {}
func (TimerFired) command()  {}
