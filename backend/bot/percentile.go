package bot

import (
	"math/bits"
	"math/rand/v2"
	"slices"
	"sort"
	"sync"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/holdem"
)

// Starting hands are ranked by percentile within their game, 1 for the best
// and near 0 for the worst, so ranges can be "the top fifth of hands" in any
// game. Two card hands look theirs up in a chart (chart.go). Omaha hands rank
// by their omaha score over a sample of hands.

var (
	omahaMu     sync.Mutex
	omahaScores = map[holdem.Game][]float64{} // sorted scores of sampled hands
)

// percentile is hole's percentile among starting hands in g.
func percentile(g holdem.Game, hole []eval.Card) float64 {
	if len(hole) <= 2 {
		hi, lo, suited := classOf(hole)
		chart := &fullChart
		if g.Short {
			chart = &shortChart
		}
		if !suited {
			hi, lo = lo, hi // offsuit hands sit below the pairs
		}
		return float64(chart[12-hi][12-lo]) / 1000
	}
	scores := omahaScoresOf(g)
	s := omaha(hole)
	lo := sort.SearchFloat64s(scores, s)
	hi := sort.Search(len(scores), func(i int) bool { return scores[i] > s })
	return (float64(lo) + float64(hi)) / 2 / float64(len(scores))
}

// classOf is a two card hand's high rank, low rank and whether it's suited.
func classOf(hole []eval.Card) (hi, lo int, suited bool) {
	hi, lo = rank(hole[0]), rank(hole[1])
	if lo > hi {
		hi, lo = lo, hi
	}
	return hi, lo, hole[0]&hole[1]&0xF000 != 0
}

func omahaScoresOf(g holdem.Game) []float64 {
	key := holdem.Game{Hole: g.HoleCards(), Short: g.Short}
	omahaMu.Lock()
	defer omahaMu.Unlock()
	if scores, ok := omahaScores[key]; ok {
		return scores
	}
	r := rand.New(rand.NewPCG(7, 7))
	deck := holdem.Deck(key)
	scores := make([]float64, 0, 20000)
	for range 20000 {
		r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
		scores = append(scores, omaha(deck[:key.Hole]))
	}
	slices.Sort(scores)
	omahaScores[key] = scores
	return scores
}

// cardBit is c's bit in a set of cards.
func cardBit(c eval.Card) uint64 {
	return 1 << (rank(c)*4 + bits.TrailingZeros32(uint32(c>>12&0xF)))
}

func maskOf(cards []eval.Card) uint64 {
	var m uint64
	for _, c := range cards {
		m |= cardBit(c)
	}
	return m
}
