package holdem

import (
	"slices"
	"strings"

	"github.com/alexclewontin/riverboat/eval"
)

// Game is what's being played. The zero Game is no-limit hold'em; more than
// two hole cards is Omaha.
type Game struct {
	Hole  int  `json:"hole"`  // hole cards each, 0 for 2
	Pot   bool `json:"pot"`   // pot limit
	Short bool `json:"short"` // short deck: 2s to 5s out, a flush beats a full house, trips beat a straight, A-6-7-8-9 is a straight
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
	if g.Short {
		return slices.DeleteFunc(slices.Clone(eval.DefaultDeck), func(c Card) bool { return rank(c) < 4 })
	}
	return slices.Clone(eval.DefaultDeck)
}

// rank is c's rank, 0 for a deuce to 12 for an ace.
func rank(c Card) int { return int(c>>8) & 0xF }

// Best is the best five card hand hole makes with board, and its score,
// lower is better (Cactus Kev's ranks). Hold'em plays any five of them, Omaha
// exactly two hole cards and three from the board. Without enough cards it's
// nil and a score worse than any hand.
func (g Game) Best(hole, board []Card) ([]Card, int) {
	if g.Short {
		return shortBest(g, hole, board)
	}
	if g.HoleCards() > 2 {
		return omaha(hole, board, eval.HandValue)
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

// omaha is the best hand of two of hole and three of board, scored by value.
func omaha(hole, board []Card, value func(a, b, c, d, e Card) int) ([]Card, int) {
	var best []Card
	score := worstScore + 1
	for i := range hole {
		for j := i + 1; j < len(hole); j++ {
			for a := range board {
				for b := a + 1; b < len(board); b++ {
					for c := b + 1; c < len(board); c++ {
						if s := value(hole[i], hole[j], board[a], board[b], board[c]); s < score {
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
	return strings.TrimPrefix(g.HandName(score, nil), "a ")
}

// HandName names a hand from its score in g, like "a flush, ace high". See
// the package HandName.
func (g Game) HandName(score int, cards []Card) string {
	if g.Short {
		score = fullScore(score)
	}
	return HandName(score, cards)
}

// Cactus Kev's scores: the best of each kind of hand.
const (
	kevQuads     = 11
	kevFullHouse = 167
	kevFlush     = 323
	kevStraight  = 1600
	kevTrips     = 1610
	kevTwoPair   = 2468
)

// shortScore moves a five card hand's Cactus Kev score into short deck order:
// flushes ahead of full houses and trips ahead of straights, each kind
// keeping its order. A-6-7-8-9 takes the nine high straight's place, which no
// short deck can make otherwise.
func shortScore(a, b, c, d, e Card) int {
	s := eval.HandValue(a, b, c, d, e)
	if lowStraight(a, b, c, d, e) {
		// five ranks: a flush or high card
		if s < kevStraight {
			s = 6 // nine high straight flush
		} else {
			s = 1605 // nine high straight
		}
	}
	switch {
	case s >= kevFlush && s < kevStraight:
		return s - (kevFlush - kevFullHouse)
	case s >= kevFullHouse && s < kevFlush:
		return s + (kevStraight - kevFlush)
	case s >= kevTrips && s < kevTwoPair:
		return s - (kevTrips - kevStraight)
	case s >= kevStraight && s < kevTrips:
		return s + (kevTwoPair - kevTrips)
	}
	return s
}

// fullScore undoes shortScore, for naming.
func fullScore(s int) int {
	flushes, straights := kevStraight-kevFlush, kevTrips-kevStraight
	switch {
	case s >= kevFullHouse && s < kevFullHouse+flushes:
		return s + (kevFlush - kevFullHouse)
	case s >= kevFullHouse+flushes && s < kevStraight:
		return s - (kevStraight - kevFlush)
	case s >= kevStraight && s < kevTwoPair-straights:
		return s + (kevTrips - kevStraight)
	case s >= kevTwoPair-straights && s < kevTwoPair:
		return s - (kevTwoPair - kevTrips)
	}
	return s
}

// lowStraight is whether the five cards are A-6-7-8-9.
func lowStraight(cards ...Card) bool {
	var mask int
	for _, c := range cards {
		mask |= 1 << rank(c)
	}
	return mask == 1<<12|1<<7|1<<6|1<<5|1<<4
}

// shortBest is Best on a short deck: every five a hand can play, scored in
// short deck order.
func shortBest(g Game, hole, board []Card) ([]Card, int) {
	if g.HoleCards() > 2 {
		return omaha(hole, board, shortScore)
	}
	c := append(slices.Clone(hole), board...)
	var best []Card
	score := worstScore + 1
	try := func(five []Card) {
		if s := shortScore(five[0], five[1], five[2], five[3], five[4]); s < score {
			best, score = five, s
		}
	}
	switch len(c) {
	case 5:
		try(c)
	case 6:
		for i := range c {
			try(without(c, i))
		}
	case 7:
		for i := range c {
			for j := i + 1; j < len(c); j++ {
				try(without(c, i, j))
			}
		}
	}
	return best, score
}

// without is c less the cards at skip.
func without(c []Card, skip ...int) []Card {
	var out []Card
	for i, card := range c {
		if !slices.Contains(skip, i) {
			out = append(out, card)
		}
	}
	return out
}
