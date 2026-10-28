package holdem

import (
	"math/rand/v2"
	"slices"
	"testing"
)

var short = Game{Short: true}

func TestShortDeck(t *testing.T) {
	d := Deck(short)
	if len(d) != 36 {
		t.Fatalf("%d cards", len(d))
	}
	for _, c := range d {
		if rank(c) < 4 {
			t.Fatalf("%v in a short deck", c)
		}
	}
	a, b := shuffle(Deck(short), [32]byte{1}, 3), shuffle(Deck(short), [32]byte{1}, 3)
	if !slices.Equal(a, b) {
		t.Fatal("same seed, different decks")
	}
}

func TestShortRanks(t *testing.T) {
	// each hand beats the next
	order := []struct{ hole, board, name string }{
		{"Ts 9s", "8s 7s 6s Kd 2c", "a straight flush, ten high"},
		{"As 9s", "8s 7s 6s Kd 2c", "a straight flush, nine high"},
		{"Ah Ad", "Ac As Kd 7c 6h", "four of a kind, aces"},
		{"Ah Kh", "Qh 9h 7h Kd 6c", "a flush, ace high"},
		{"Ah Ad", "Ac Kd Ks 7c 6h", "a full house, aces full of kings"},
		{"Ah Ad", "Ac Jd Ts 7c 6h", "three of a kind, aces"},
		{"Ah Kd", "Qc Jd Ts 7c 6h", "a straight, ace high"},
		{"Tc 9d", "8c 7d 6s Kc Ks", "a straight, ten high"},
		{"Ac 9d", "8c 7d 6s Kc Qs", "a straight, nine high"},
		{"Ah Ad", "Kc Kd Ts 7c 6h", "two pair, aces and kings"},
	}
	prev := 0
	for i, h := range order {
		cards, score := short.Best(cards(h.hole), cards(h.board))
		if got := short.HandName(score, cards); got != h.name {
			t.Errorf("%s %s: %q, want %q", h.hole, h.board, got, h.name)
		}
		if i > 0 && score <= prev {
			t.Errorf("%s doesn't lose to %s: %d <= %d", h.name, order[i-1].name, score, prev)
		}
		prev = score
	}
}

// Best with Short agrees with ranking every five by hand, and names round trip
func TestShortBestRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for range 2000 {
		d := Deck(short)
		r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
		_, score := short.Best(d[:2], d[2:7])
		want := worstScore + 1
		for i := range 7 {
			for j := i + 1; j < 7; j++ {
				f := without(d[:7], i, j)
				want = min(want, shortScore(f[0], f[1], f[2], f[3], f[4]))
			}
		}
		if score != want {
			t.Fatalf("%v scored %d, want %d", d[:7], score, want)
		}
		if fullScore(score) < 1 || fullScore(score) > worstScore || short.HandName(score, nil) == "" {
			t.Fatalf("score %d names %q", score, short.HandName(score, nil))
		}
	}
	for s := 1; s <= worstScore; s++ {
		if fullScore(shortScoreOf(s)) != s {
			t.Fatalf("Kev score %d doesn't round trip", s)
		}
	}
}

// shortScoreOf is shortScore's remapping for a Cactus Kev score.
func shortScoreOf(s int) int {
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

func TestShortOmaha(t *testing.T) {
	g := Game{Hole: 4, Short: true}
	// trips beat the broadway straight, and one heart in the hole is no flush
	five, score := g.Best(cards("Ah Kd Kc 6s"), cards("Qh Jh Th Ks 9h"))
	if got := g.HandName(score, nil); got != "three of a kind" {
		t.Fatalf("%v is %q", five, got)
	}
	if _, score := g.Best(cards("Ah 6d Kc Ks"), cards("7h 8c 9s Qd Jd")); g.HandName(score, nil) != "a straight" {
		t.Fatalf("A-6 with 7-8-9 is %q", g.HandName(score, nil))
	}
}

func TestShortShowdown(t *testing.T) {
	// seat 1's flush beats seat 2's full house
	stackDraws(t, "Ah 9h", "Kd Kc", "Kh 7h 6h", "7d", "Ts")
	s := game(t, 1000, 1000)
	s.Config.Game = short
	start(t, s)
	act(t, s, 1, Call{})
	act(t, s, 2, Check{})
	for s.InHand() {
		if s.Betting {
			act(t, s, s.ToAct, Check{})
		} else {
			street(t, s)
		}
	}
	if s.Player(1).Stack != 1020 || s.Pots[0].Runs[0].Hand != "a flush, ace high" {
		t.Fatalf("stacks %v, %+v", stacks(s), s.Pots[0].Runs[0])
	}
}
