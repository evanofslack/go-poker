package server

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/evanofslack/go-poker/bot"
)

// input length limits, mirrored by maxLength on the frontend inputs
const (
	maxTablenameLen = 32
	maxUsernameLen  = 20
	maxMessageLen   = 200
	maxPasswordLen  = 32
	maxReasonLen    = 60
	maxChipChange   = 1_000_000_000
)

// table settings limits, mirrored by the create table form
const (
	maxSeats         = 9
	maxBigBlind      = 1_000_000
	maxBuyInBB       = 1000 // max buy-in, in big blinds
	minTurnTime      = 10_000
	maxTurnTime      = 300_000
	minNextHandDelay = 2000
	maxNextHandDelay = 15_000
)

// validSettings returns why s can't be used for a new table, if it can't.
func validSettings(s tableSettings) error {
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

// validUsername reports whether s can be used as a player's name.
func validUsername(s string) bool {
	return validLength(s, maxUsernameLen) && !strings.EqualFold(strings.TrimSpace(s), gameAdminName)
}

// validLength reports whether s is non-blank and at most max characters.
func validLength(s string, max int) bool {
	n := utf8.RuneCountInString(s)
	return strings.TrimSpace(s) != "" && n <= max
}

// validPassword reports whether s can be a table's password. Empty means none.
func validPassword(s string) bool {
	return utf8.RuneCountInString(s) <= maxPasswordLen
}

// validAdjust reports whether n is a usable host chip change: not zero, and
// not absurdly large either way.
func validAdjust(n int64) bool {
	return n != 0 && n >= -maxChipChange && n <= maxChipChange
}

// validReason reports whether s can be the reason for a chip change. It's optional.
func validReason(s string) bool {
	return utf8.RuneCountInString(s) <= maxReasonLen
}

// validAmount reports whether n is a usable chip amount, like a rebuy.
func validAmount(n uint) bool {
	return n > 0
}

// validBotLevel reports whether s is a level a bot can play at.
func validBotLevel(s string) bool {
	return bot.Valid(s)
}
