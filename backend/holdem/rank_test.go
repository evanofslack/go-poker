package holdem

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
)

func TestBestHoldem(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var g Game
	for range 2000 {
		d := Deck(g)
		r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
		wantCards, want := eval.BestFiveOfSeven(d[0], d[1], d[2], d[3], d[4], d[5], d[6])
		cards, score := g.Best(d[:2], d[2:7])
		if score != want || !slices.Equal(cards, wantCards) {
			t.Fatalf("Best(%v, %v) = %v %d, want %v %d", d[:2], d[2:7], cards, score, wantCards, want)
		}
		if _, score := g.Best(d[:2], d[2:6]); score != best6(d[:6]) {
			t.Fatalf("six cards %v scored %d", d[:6], score)
		}
		if _, score := g.Best(d[:2], d[2:5]); score != eval.HandValue(d[0], d[1], d[2], d[3], d[4]) {
			t.Fatalf("five cards %v scored %d", d[:5], score)
		}
	}
}

func best6(c []Card) int {
	_, s := eval.BestFiveOfSix(c[0], c[1], c[2], c[3], c[4], c[5])
	return s
}

func TestBestShortBoard(t *testing.T) {
	d := Deck(Game{})
	if cards, score := (Game{}).Best(d[:2], nil); cards != nil || score <= worstScore {
		t.Fatalf("no board = %v %d, want nil and worse than any hand", cards, score)
	}
}

func TestHoleCards(t *testing.T) {
	if n := (Game{}).HoleCards(); n != 2 {
		t.Fatalf("hold'em deals %d", n)
	}
	s := game(t, 1000, 1000, 1000)
	if _, err := s.StartHand(0); err != nil {
		t.Fatal(err)
	}
	for _, p := range s.Players {
		if len(p.Cards) != 2 {
			t.Fatalf("seat %d dealt %v", p.Seat, p.Cards)
		}
	}
	v := s.View(1)
	if len(v.Players[1].Cards) != 2 || !hidden(v.Players[1].Cards) || hidden(v.Players[0].Cards) {
		t.Fatalf("view hides wrong: %v %v", v.Players[0].Cards, v.Players[1].Cards)
	}
}

func TestViewUndealt(t *testing.T) {
	s := game(t, 1000, 1000)
	for _, p := range s.View(1).Players {
		if p.Cards == nil {
			t.Fatalf("seat %d cards nil before a hand", p.Seat)
		}
	}
}

func TestCloneCards(t *testing.T) {
	s := game(t, 1000, 1000)
	if _, err := s.StartHand(0); err != nil {
		t.Fatal(err)
	}
	c := s.Clone()
	c.Players[0].Cards[0] = 0
	if s.Players[0].Cards[0] == 0 {
		t.Fatal("clone shares hole cards")
	}
}

func TestMadeHand(t *testing.T) {
	var g Game
	tests := []struct {
		hole, board string
		want        string
	}{
		{"As Ad", "", "pair"},
		{"As Kd", "", "high card"},
		{"As Kd", "Ah 7c 2d", "pair"},
		{"As Kd", "Ah Kc 2d", "two pair"},
		{"As Ks", "Qs Js Ts", "royal flush"},
		{"9s 8s", "7d 6c 5h 2d", "straight"},
		{"9s 8s", "9d 9c 8h 2d Kc", "full house"},
	}
	for _, tt := range tests {
		var board []Card
		if tt.board != "" {
			board = cards(tt.board)
		}
		if got := g.MadeHand(cards(tt.hole), board); got != tt.want {
			t.Errorf("MadeHand(%s, %s) = %q, want %q", tt.hole, tt.board, got, tt.want)
		}
	}
	if got := g.MadeHand([]Card{0, 0}, nil); got != "" {
		t.Errorf("hidden cards made %q", got)
	}
}
