package server

import (
	"strings"
	"unicode/utf8"

	"github.com/evanofslack/go-poker/bot"
	"github.com/evanofslack/go-poker/table"
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

// validUsername reports whether s can be used as a player's name.
func validUsername(s string) bool {
	return validLength(s, maxUsernameLen) && !strings.EqualFold(strings.TrimSpace(s), table.SystemName)
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

// validHandNumber reports whether n can name a hand. Hands count from 1.
func validHandNumber(n int) bool {
	return n > 0
}

// validReason reports whether s can be the reason for a chip change. It's optional.
func validReason(s string) bool {
	return utf8.RuneCountInString(s) <= maxReasonLen
}

// validBuyIn reports whether n is a usable chip amount, like a rebuy.
func validBuyIn(n int64) bool {
	return n > 0 && n <= maxChipChange
}

// validPreset reports whether s is a straddle preset.
func validPreset(s string) bool {
	return s == table.PresetAsk || s == table.PresetAlways || s == table.PresetNever
}

// validRuns reports whether n is a pick in the run it vote.
func validRuns(n int) bool {
	return n >= 1 && n <= table.MaxRuns
}

// validBotLevel reports whether s is a level a bot can play at.
func validBotLevel(s string) bool {
	return bot.Valid(s)
}
