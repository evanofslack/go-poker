package holdem

import (
	"slices"

	"github.com/alexclewontin/riverboat/eval"
)

// Game is what's being played. The zero Game is hold'em.
type Game struct {
	Hole int `json:"hole"` // hole cards each, 0 for 2
}

// HoleCards is how many hole cards each player gets.
func (g Game) HoleCards() int {
	if g.Hole == 0 {
		return 2
	}
	return g.Hole
}

// Deck is the game's cards, in order.
func Deck(g Game) []Card {
	return slices.Clone(eval.DefaultDeck)
}

// Best is the best five card hand hole makes with board, and its score,
// lower is better (Cactus Kev's ranks). Hold'em plays any five of them. With
// fewer than five cards it's nil and a score worse than any hand.
func (g Game) Best(hole, board []Card) ([]Card, int) {
	c := append(slices.Clone(hole), board...)
	switch len(c) {
	case 5:
		return slices.Clone(c), eval.HandValue(c[0], c[1], c[2], c[3], c[4])
	case 6:
		return eval.BestFiveOfSix(c[0], c[1], c[2], c[3], c[4], c[5])
	case 7:
		return eval.BestFiveOfSeven(c[0], c[1], c[2], c[3], c[4], c[5], c[6])
	}
	return nil, worstScore + 1
}
