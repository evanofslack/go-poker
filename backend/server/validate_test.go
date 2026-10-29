package server

import (
	"strings"
	"testing"
)

func TestValidLength(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want bool
	}{
		{"evan", maxUsernameLen, true},
		{"", maxUsernameLen, false},
		{"   ", maxUsernameLen, false},
		{strings.Repeat("a", maxUsernameLen), maxUsernameLen, true},
		{strings.Repeat("a", maxUsernameLen+1), maxUsernameLen, false},
		{strings.Repeat("é", maxUsernameLen), maxUsernameLen, true}, // counts characters, not bytes
	}
	for _, tt := range tests {
		if got := validLength(tt.s, tt.max); got != tt.want {
			t.Errorf("validLength(%q, %d) = %v, want %v", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestValidUsername(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"evan", true},
		{"", false},
		{strings.Repeat("a", maxUsernameLen+1), false},
		{"system", false},
		{"System", false},
		{" SYSTEM ", false},
		{"systems", true},
	}
	for _, tt := range tests {
		if got := validUsername(tt.s); got != tt.want {
			t.Errorf("validUsername(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestValidBuyIn(t *testing.T) {
	for n, want := range map[int64]bool{0: false, -5: false, 500: true, maxChipChange + 1: false} {
		if validBuyIn(n) != want {
			t.Errorf("validBuyIn(%d) = %v, want %v", n, !want, want)
		}
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", true}, // no password
		{"hunter2", true},
		{strings.Repeat("é", maxPasswordLen), true},
		{strings.Repeat("a", maxPasswordLen+1), false},
	}
	for _, tt := range tests {
		if got := validPassword(tt.s); got != tt.want {
			t.Errorf("validPassword(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestValidBotLevel(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"easy", true},
		{"medium", true},
		{"", false},
		{"Easy", false},
		{"hard", true},
		{"expert", false},
	}
	for _, tt := range tests {
		if got := validBotLevel(tt.s); got != tt.want {
			t.Errorf("validBotLevel(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestValidAdjust(t *testing.T) {
	for n, want := range map[int64]bool{0: false, 1: true, -1: true, maxChipChange: true, -maxChipChange: true, maxChipChange + 1: false, -maxChipChange - 1: false} {
		if got := validAdjust(n); got != want {
			t.Errorf("validAdjust(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestValidReason(t *testing.T) {
	if !validReason("") || !validReason("miscount") || validReason(strings.Repeat("a", maxReasonLen+1)) {
		t.Error("validReason allows empty and short reasons only")
	}
}

func TestValidHandNumber(t *testing.T) {
	for n, want := range map[int]bool{-1: false, 0: false, 1: true, 500: true} {
		if got := validHandNumber(n); got != want {
			t.Errorf("validHandNumber(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestValidPreset(t *testing.T) {
	for s, want := range map[string]bool{"ask": true, "always": true, "never": true, "": false, "maybe": false} {
		if got := validPreset(s); got != want {
			t.Errorf("validPreset(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestValidRuns(t *testing.T) {
	for n, want := range map[int]bool{-1: false, 0: false, 1: true, 4: true, 5: false} {
		if got := validRuns(n); got != want {
			t.Errorf("validRuns(%d) = %v, want %v", n, got, want)
		}
	}
}
