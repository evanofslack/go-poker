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

// validLength reports whether s is non-blank and at most max characters.
func validLength(s string, max int) bool {
	n := utf8.RuneCountInString(s)
	return strings.TrimSpace(s) != "" && n <= max
}
