package table

import (
	"errors"
	"fmt"
)

// Settings are picked by the host when creating a table, and fixed after.
// Buy-ins are in chips, times in milliseconds.
type Settings struct {
	Seats         uint `json:"seats"`
	SmallBlind    uint `json:"sb"`
	BigBlind      uint `json:"bb"`
	MinBuy        uint `json:"minBuy"`
	MaxBuy        uint `json:"maxBuy"`
	TurnTime      uint `json:"turnTime"`      // time to act before checking or folding for you, 0 for no limit
	NextHandDelay uint `json:"nextHandDelay"` // pause after a hand before dealing the next
	Public        bool `json:"public"`        // listed on the home page
	ApproveRebuys bool `json:"approveRebuys"` // rebuys, and sitting back down, wait for the host
}

// DefaultSettings are used for anything create-table leaves out.
var DefaultSettings = Settings{
	Seats:         8,
	SmallBlind:    10,
	BigBlind:      20,
	MinBuy:        1000,
	MaxBuy:        4000,
	TurnTime:      30000,
	NextHandDelay: 5000,
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
	case s.TurnTime != 0 && (s.TurnTime < minTurnTime || s.TurnTime > maxTurnTime):
		return fmt.Errorf("turn timer must be off or between %d and %d seconds", minTurnTime/1000, maxTurnTime/1000)
	case s.NextHandDelay < minNextHandDelay || s.NextHandDelay > maxNextHandDelay:
		return fmt.Errorf("next hand delay must be between %d and %d seconds", minNextHandDelay/1000, maxNextHandDelay/1000)
	}
	return nil
}
