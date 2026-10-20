package server

import (
	"strings"
	"unicode/utf8"
)

// input length limits, mirrored by maxLength on the frontend inputs
const (
	maxTablenameLen = 32
	maxUsernameLen  = 20
	maxMessageLen   = 200
)

// validUsername reports whether s can be used as a player's name.
func validUsername(s string) bool {
	return validLength(s, maxUsernameLen) && !strings.EqualFold(strings.TrimSpace(s), gameAdminName)
}

// validLength reports whether s is non-blank and at most max characters.
func validLength(s string, max int) bool {
	n := utf8.RuneCountInString(s)
	return strings.TrimSpace(s) != "" && n <= max
}

// validAmount reports whether n is a usable chip amount, like a rebuy.
func validAmount(n uint) bool {
	return n > 0
}
