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

func TestValidAmount(t *testing.T) {
	if validAmount(0) {
		t.Error("validAmount(0) = true, want false")
	}
	if !validAmount(500) {
		t.Error("validAmount(500) = false, want true")
	}
}
