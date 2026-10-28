package holdem

import (
	"slices"
	"strings"

	"github.com/alexclewontin/riverboat/eval"
)

// Game is what's being played. The zero Game is no-limit hold'em; more than
// two hole cards is Omaha.
type Game struct {
	Hole int  `json:"hole"` // hole cards each, 0 for 2
	Pot  bool `json:"pot"`  // pot limit
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
// lower is better (Cactus Kev's ranks). Hold'em plays any five of them, Omaha
// exactly two hole cards and three from the board. Without enough cards it's
// nil and a score worse than any hand.
func (g Game) Best(hole, board []Card) ([]Card, int) {
	if g.HoleCards() > 2 {
		return omaha(hole, board)
	}
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

// omaha is the best hand of two of hole and three of board.
func omaha(hole, board []Card) ([]Card, int) {
	var best []Card
	score := worstScore + 1
	for i := range hole {
		for j := i + 1; j < len(hole); j++ {
			for a := range board {
				for b := a + 1; b < len(board); b++ {
					for c := b + 1; c < len(board); c++ {
						if s := eval.HandValue(hole[i], hole[j], board[a], board[b], board[c]); s < score {
							best, score = []Card{hole[i], hole[j], board[a], board[b], board[c]}, s
						}
					}
				}
			}
		}
	}
	return best, score
}

// MadeHand is the kind of hand hole makes with board so far, like "two pair",
// for a player's own hint. Before the flop it's a pair in the hole or high
// card. Empty without cards.
func (g Game) MadeHand(hole, board []Card) string {
	if len(hole) == 0 || hole[0] == 0 {
		return ""
	}
	if len(board) < 3 {
		seen := map[byte]bool{}
		for _, c := range hole {
			r := c.String()[0]
			if seen[r] {
				return "pair"
			}
			seen[r] = true
		}
		return "high card"
	}
	_, score := g.Best(hole, board)
	return strings.TrimPrefix(HandName(score, nil), "a ")
}
