package bot

import (
	"hash/fnv"
	"math/rand/v2"
)

// style is how one medium bot leans, so a table of them doesn't play alike.
type style struct {
	loose float64 // added to its starting hand scores, from -1 to 1.5
	aggro float64 // how much more often it bluffs and bets without a hand, from -0.3 to 0.3
}

// styleOf is the style of the player with uuid, the same every hand.
func styleOf(uuid string) style {
	h := fnv.New64a()
	h.Write([]byte(uuid))
	r := rand.New(rand.NewPCG(h.Sum64(), 0))
	return style{loose: 2.5*r.Float64() - 1, aggro: 0.6*r.Float64() - 0.3}
}

// often reports true with chance p, more or less often as the bot's style
// leans.
func (s spot) often(p float64, r *rand.Rand) bool {
	return r.Float64() < p*(1+s.style.aggro)
}
