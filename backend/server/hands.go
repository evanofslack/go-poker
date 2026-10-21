package server

import (
	"cmp"
	"slices"
	"strings"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/poker"
)

// ranks are card ranks from low to high, as eval.Card.String writes them
const ranks = "23456789TJQKA"

var rankNames = [...]string{"two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "jack", "queen", "king", "ace"}

// plural names more than one card of a rank, like "sixes"
func plural(rank int) string {
	if rank == 4 { // six
		return "sixes"
	}
	return rankNames[rank] + "s"
}

// handName names a five card hand from its score, where lower is better
// (Cactus Kev's ranks), like "two pair, kings and sevens". A hand won without
// a showdown scores 8000 and has no name. Without the cards it names only the
// kind of hand.
func handName(score int, cards []eval.Card) string {
	if score <= 0 || score > 7462 {
		return ""
	}

	// ranks by how many of each there are, then high to low
	counts := map[int]int{}
	for _, c := range cards {
		counts[strings.IndexByte(ranks, c.String()[0])]++
	}
	var r []int
	for rank := range counts {
		r = append(r, rank)
	}
	slices.SortFunc(r, func(a, b int) int {
		return cmp.Or(counts[b]-counts[a], b-a)
	})
	named := len(cards) == 5 && len(counts) > 0 && r[len(r)-1] >= 0
	// a straight's high card, the five in a wheel
	high := func() string {
		if counts[12] == 1 && counts[0] == 1 && counts[11] == 0 {
			return rankNames[3]
		}
		return rankNames[r[0]]
	}

	switch {
	case score == 1:
		return "a royal flush"
	case score <= 10:
		if !named {
			return "a straight flush"
		}
		return "a straight flush, " + high() + " high"
	case score <= 166:
		if !named {
			return "four of a kind"
		}
		return "four of a kind, " + plural(r[0])
	case score <= 322:
		if !named {
			return "a full house"
		}
		return "a full house, " + plural(r[0]) + " full of " + plural(r[1])
	case score <= 1599:
		if !named {
			return "a flush"
		}
		return "a flush, " + rankNames[r[0]] + " high"
	case score <= 1609:
		if !named {
			return "a straight"
		}
		return "a straight, " + high() + " high"
	case score <= 2467:
		if !named {
			return "three of a kind"
		}
		return "three of a kind, " + plural(r[0])
	case score <= 3325:
		if !named {
			return "two pair"
		}
		return "two pair, " + plural(r[0]) + " and " + plural(r[1])
	case score <= 6185:
		if !named {
			return "a pair"
		}
		return "a pair of " + plural(r[0])
	default:
		if !named {
			return "high card"
		}
		return rankNames[r[0]] + " high"
	}
}

// potHands names the hand that won each pot, empty for one won without a
// showdown
func potHands(pots []poker.Pot) []string {
	names := make([]string, len(pots))
	for i, pot := range pots {
		if len(pot.WinningPlayerNums) > 0 {
			names[i] = handName(pot.WinningScore, pot.WinningHand)
		}
	}
	return names
}
