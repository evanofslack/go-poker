package holdem

import (
	"cmp"
	"slices"
	"strings"
)

// worstScore is the score of the worst five card hand, 7-5-4-3-2 offsuit.
const worstScore = 7462

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

// HandName names a five card hand from its score, where lower is better
// (Cactus Kev's ranks), like "two pair, kings and sevens". A pot won without
// a showdown scores 0 and has no name. Without the cards it names only the
// kind of hand.
func HandName(score int, cards []Card) string {
	if score <= 0 || score > worstScore {
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
	// a straight's high card, the five in a wheel and the nine in A-6-7-8-9
	high := func() string {
		if counts[12] == 1 && counts[0] == 1 && counts[11] == 0 {
			return rankNames[3]
		}
		// short deck's A-6-7-8-9
		if counts[12] == 1 && counts[4] == 1 && counts[8] == 0 {
			return rankNames[7]
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
