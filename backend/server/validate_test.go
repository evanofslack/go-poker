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

func TestValidSettings(t *testing.T) {
	ok := tableSettings{Seats: 8, SmallBlind: 10, BigBlind: 20, MinBuy: 1000, MaxBuy: 4000, TurnTime: 30000, NextHandDelay: 5000}
	tests := []struct {
		name  string
		edit  func(*tableSettings)
		valid bool
	}{
		{"defaults", func(*tableSettings) {}, true},
		{"heads up", func(s *tableSettings) { s.Seats = 2 }, true},
		{"nine seats", func(s *tableSettings) { s.Seats = 9 }, true},
		{"one seat", func(s *tableSettings) { s.Seats = 1 }, false},
		{"ten seats", func(s *tableSettings) { s.Seats = 10 }, false},
		{"equal blinds", func(s *tableSettings) { s.SmallBlind = 20 }, true},
		{"no small blind", func(s *tableSettings) { s.SmallBlind = 0 }, false},
		{"small above big", func(s *tableSettings) { s.SmallBlind = 21 }, false},
		{"huge big blind", func(s *tableSettings) { s.BigBlind = maxBigBlind + 1 }, false},
		{"min of one big blind", func(s *tableSettings) { s.MinBuy = 20 }, true},
		{"min under a big blind", func(s *tableSettings) { s.MinBuy = 19 }, false},
		{"min above max", func(s *tableSettings) { s.MinBuy = 4001 }, false},
		{"max at the cap", func(s *tableSettings) { s.MaxBuy = 20 * maxBuyInBB }, true},
		{"max over the cap", func(s *tableSettings) { s.MaxBuy = 20*maxBuyInBB + 1 }, false},
		{"no turn timer", func(s *tableSettings) { s.TurnTime = 0 }, true},
		{"turn too short", func(s *tableSettings) { s.TurnTime = minTurnTime - 1 }, false},
		{"turn too long", func(s *tableSettings) { s.TurnTime = maxTurnTime + 1 }, false},
		{"no hand delay", func(s *tableSettings) { s.NextHandDelay = 0 }, false},
		{"hand delay too long", func(s *tableSettings) { s.NextHandDelay = maxNextHandDelay + 1 }, false},
	}
	for _, tt := range tests {
		s := ok
		tt.edit(&s)
		if err := validSettings(s); (err == nil) != tt.valid {
			t.Errorf("%s: validSettings = %v, want valid %v", tt.name, err, tt.valid)
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
