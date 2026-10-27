package table

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evanofslack/go-poker/holdem"
)

// Settings are picked by the host when creating a table. The host can change
// them later, all but the seat count.
// Buy-ins are in chips, times in milliseconds.
type Settings struct {
	Seats         uint `json:"seats"`
	SmallBlind    uint `json:"sb"`
	BigBlind      uint `json:"bb"`
	MinBuy        uint `json:"minBuy"`
	MaxBuy        uint `json:"maxBuy"`
	TurnTime      uint `json:"turnTime"`      // time to act before checking or folding for you, 0 for no limit
	NextHandDelay uint `json:"nextHandDelay"` // pause after a hand before dealing the next
	DecisionTime  uint `json:"decisionTime"`  // to answer a straddle ask or a run-it vote
	Public        bool `json:"public"`        // listed on the home page
	ApproveRebuys bool `json:"approveRebuys"` // rebuys, and sitting back down, wait for the host

	// house rules
	Ante     uint `json:"ante"`     // from every player each hand, 0 for none
	DuckHunt bool `json:"duckHunt"` // players can turn up the rest of the board after a hand ends early
	// SevenDeuce is the bounty everyone dealt in pays a player who wins the
	// main pot with 7-2 and shows it, 0 for none
	SevenDeuce uint   `json:"sevenDeuce"`
	Straddle   string `json:"straddle"` // StraddleOff, StraddleOptional or StraddleForced
	RunIt      bool   `json:"runIt"`    // once everyone left is all in, or all but one, they vote to run the rest of the board out more than once
}

// Straddle settings. Forced posts under the gun's straddle for them;
// re-straddles stay optional.
const (
	StraddleOff      = "off"
	StraddleOptional = "optional"
	StraddleForced   = "forced"
)

// DefaultSettings are used for anything create-table leaves out.
var DefaultSettings = Settings{
	Seats:         8,
	SmallBlind:    10,
	BigBlind:      20,
	MinBuy:        1000,
	MaxBuy:        4000,
	TurnTime:      30000,
	NextHandDelay: 5000,
	DecisionTime:  5000,
	DuckHunt:      true,
	RunIt:         true,
	Straddle:      StraddleOff,
}

// settings limits, mirrored by the create table form
const (
	maxSeats         = 9
	maxBigBlind      = 1_000_000
	maxBuyInBB       = 1000 // max buy-in, in big blinds
	minTurnTime      = 10_000
	maxTurnTime      = 300_000
	minNextHandDelay = 2000
	maxNextHandDelay = 15_000
	minDecisionTime  = 2000
	maxDecisionTime  = 30_000
	maxSevenDeuceBB  = 10 // the most a 7-2 bounty can be, in big blinds
)

// Validate returns why s can't be used for a new table, if it can't.
func (s Settings) Validate() error {
	switch {
	case s.Seats < 2 || s.Seats > maxSeats:
		return fmt.Errorf("seats must be between 2 and %d", maxSeats)
	case s.SmallBlind < 1 || s.SmallBlind > s.BigBlind:
		return errors.New("small blind must be between 1 and the big blind")
	case s.BigBlind > maxBigBlind:
		return fmt.Errorf("big blind can't be more than %d", maxBigBlind)
	case s.MinBuy < s.BigBlind || s.MinBuy > s.MaxBuy:
		return errors.New("min buy-in must be between 1 big blind and the max buy-in")
	case s.MaxBuy > maxBuyInBB*s.BigBlind:
		return fmt.Errorf("max buy-in can't be more than %d big blinds", maxBuyInBB)
	case s.Ante > s.BigBlind:
		return errors.New("ante can't be more than the big blind")
	case s.SevenDeuce > maxSevenDeuceBB*s.BigBlind:
		return fmt.Errorf("7-2 bounty can't be more than %d big blinds", maxSevenDeuceBB)
	case s.TurnTime != 0 && (s.TurnTime < minTurnTime || s.TurnTime > maxTurnTime):
		return fmt.Errorf("turn timer must be off or between %d and %d seconds", minTurnTime/1000, maxTurnTime/1000)
	case s.NextHandDelay < minNextHandDelay || s.NextHandDelay > maxNextHandDelay:
		return fmt.Errorf("next hand delay must be between %d and %d seconds", minNextHandDelay/1000, maxNextHandDelay/1000)
	case s.DecisionTime < minDecisionTime || s.DecisionTime > maxDecisionTime:
		return fmt.Errorf("decision time must be between %d and %d seconds", minDecisionTime/1000, maxDecisionTime/1000)
	case s.Straddle != StraddleOff && s.Straddle != StraddleOptional && s.Straddle != StraddleForced:
		return errors.New("straddle must be off, optional or forced")
	}
	return nil
}

// updateSettings changes the settings now between hands, or from the next
// hand mid-hand.
func (r *reducer) updateSettings(p *Person, set Settings) error {
	s := r.s
	if p.Name != s.Host {
		return errNotHost
	}
	set.Seats = s.Settings.Seats
	if err := set.Validate(); err != nil {
		return fail("%s", err)
	}
	if !s.Game.InHand() {
		r.applySettings(p.Name+" changed", set)
		return nil
	}
	changes := settingsChanges(s.Settings, set)
	if changes == "" {
		if s.Next != nil {
			s.Next = nil
			r.log("%s undid the settings changes for the next hand", p.Name)
		}
		return nil
	}
	s.Next = &set
	r.log("%s changed %s, from the next hand", p.Name, changes)
	return nil
}

// applySettings switches to set between hands, logging what changed after who.
func (r *reducer) applySettings(who string, set Settings) {
	s := r.s
	changes := settingsChanges(s.Settings, set)
	if s.Settings.SmallBlind != set.SmallBlind || s.Settings.BigBlind != set.BigBlind || s.Settings.Ante != set.Ante {
		r.dropStraddles() // asked at the old blinds
	}
	s.Settings, s.Next = set, nil
	s.Game.SetConfig(holdem.Config{SmallBlind: set.SmallBlind, BigBlind: set.BigBlind, Ante: set.Ante})
	if changes != "" {
		r.log("%s %s", who, changes)
	}
	r.scheduleDeal()
}

// settingsChanges lists what changed from old to set, like "the blinds to
// 25/50, the turn timer to 60s", or "" for nothing.
func settingsChanges(old, set Settings) string {
	var c []string
	add := func(changed bool, format string, args ...any) {
		if changed {
			c = append(c, fmt.Sprintf(format, args...))
		}
	}
	onOff := func(on bool) string {
		if on {
			return "on"
		}
		return "off"
	}
	chips := func(n uint) string {
		if n == 0 {
			return "off"
		}
		return fmt.Sprint(n)
	}
	add(old.SmallBlind != set.SmallBlind || old.BigBlind != set.BigBlind, "the blinds to %d/%d", set.SmallBlind, set.BigBlind)
	add(old.Ante != set.Ante, "the ante to %s", chips(set.Ante))
	add(old.MinBuy != set.MinBuy || old.MaxBuy != set.MaxBuy, "the buy-in to %d-%d", set.MinBuy, set.MaxBuy)
	if set.TurnTime == 0 {
		add(old.TurnTime != 0, "the turn timer off")
	} else {
		add(old.TurnTime != set.TurnTime, "the turn timer to %ds", set.TurnTime/1000)
	}
	add(old.NextHandDelay != set.NextHandDelay, "the time between hands to %ds", set.NextHandDelay/1000)
	add(old.Public != set.Public, "the table to %s", visibility(set.Public))
	add(old.ApproveRebuys != set.ApproveRebuys, "rebuy approval %s", onOff(set.ApproveRebuys))
	add(old.DuckHunt != set.DuckHunt, "duck hunt %s", onOff(set.DuckHunt))
	add(old.SevenDeuce != set.SevenDeuce, "the 7-2 bounty to %s", chips(set.SevenDeuce))
	add(old.Straddle != set.Straddle, "straddles %s", set.Straddle)
	add(old.RunIt != set.RunIt, "run it %s", onOff(set.RunIt))
	add(old.DecisionTime != set.DecisionTime, "the decision time to %ds", set.DecisionTime/1000)
	return strings.Join(c, ", ")
}

func visibility(public bool) string {
	if public {
		return "public"
	}
	return "private"
}

func (r *reducer) setPassword(p *Person, password string) error {
	s := r.s
	if p.Name != s.Host {
		return errNotHost
	}
	if password == "" {
		s.Password = ""
		r.log("%s removed the password", p.Name)
	} else {
		s.Password = hashPassword(password)
		r.log("%s changed the password", p.Name)
	}
	r.send("", NewPassword{password})
	return nil
}
