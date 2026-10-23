package table

import "testing"

func TestValidSettings(t *testing.T) {
	ok := Settings{Seats: 8, SmallBlind: 10, BigBlind: 20, MinBuy: 1000, MaxBuy: 4000, TurnTime: 30000, NextHandDelay: 5000}
	tests := []struct {
		name  string
		edit  func(*Settings)
		valid bool
	}{
		{"defaults", func(*Settings) {}, true},
		{"heads up", func(s *Settings) { s.Seats = 2 }, true},
		{"nine seats", func(s *Settings) { s.Seats = 9 }, true},
		{"one seat", func(s *Settings) { s.Seats = 1 }, false},
		{"ten seats", func(s *Settings) { s.Seats = 10 }, false},
		{"equal blinds", func(s *Settings) { s.SmallBlind = 20 }, true},
		{"no small blind", func(s *Settings) { s.SmallBlind = 0 }, false},
		{"small above big", func(s *Settings) { s.SmallBlind = 21 }, false},
		{"huge big blind", func(s *Settings) { s.BigBlind = maxBigBlind + 1 }, false},
		{"min of one big blind", func(s *Settings) { s.MinBuy = 20 }, true},
		{"min under a big blind", func(s *Settings) { s.MinBuy = 19 }, false},
		{"min above max", func(s *Settings) { s.MinBuy = 4001 }, false},
		{"max at the cap", func(s *Settings) { s.MaxBuy = 20 * maxBuyInBB }, true},
		{"max over the cap", func(s *Settings) { s.MaxBuy = 20*maxBuyInBB + 1 }, false},
		{"no turn timer", func(s *Settings) { s.TurnTime = 0 }, true},
		{"turn too short", func(s *Settings) { s.TurnTime = minTurnTime - 1 }, false},
		{"turn too long", func(s *Settings) { s.TurnTime = maxTurnTime + 1 }, false},
		{"no hand delay", func(s *Settings) { s.NextHandDelay = 0 }, false},
		{"hand delay too long", func(s *Settings) { s.NextHandDelay = maxNextHandDelay + 1 }, false},
	}
	for _, tt := range tests {
		s := ok
		tt.edit(&s)
		if err := s.Validate(); (err == nil) != tt.valid {
			t.Errorf("%s: Validate = %v, want valid %v", tt.name, err, tt.valid)
		}
	}
}
