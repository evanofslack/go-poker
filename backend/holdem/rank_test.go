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
	if _, err := s.StartHand(Deal{}); err != nil {
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
	if _, err := s.StartHand(Deal{}); err != nil {
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

func TestBestOmaha(t *testing.T) {
	plo := Game{Hole: 4}
	tests := []struct {
		name, hole, board, want string
	}{
		{"one suited hole card isn't a flush", "As Kd Qc Jh", "2s 5s 8s Ts 3d", "high card"},
		{"a four card board straight doesn't play", "2c 2d 3h 3s", "Ts Jd Qc Kh 9d", "a pair"},
		{"broadway needs only two from the hole", "As Kd 4c 4h", "Qs Jd Tc 2h 7d", "a straight"},
		{"quads on the board play as trips at best", "As Ad 2h 3c", "Kc Kd Kh Ks 7d", "a full house"},
		{"two hole cards make the wheel", "As 2d Kh Kc", "3c 4d 5h Tc Js", "a straight"},
		{"one pair from the hole can't fill up", "Qc Qd 9h 9s", "Qs 9d 2c 7h 4d", "three of a kind"},
		{"a paired board fills a set", "Qc Qd 9h 8s", "Qs 7d 7c 2h 4d", "a full house"},
		{"a pair on board plays with any two", "Ac Kd 7h 3s", "8c 8d 2h 5c Js", "a pair"},
	}
	for _, tt := range tests {
		five, score := plo.Best(cards(tt.hole), cards(tt.board))
		if got := HandName(score, nil); got != tt.want {
			t.Errorf("%s: %s with %s is %q (%v), want %q", tt.name, tt.hole, tt.board, got, five, tt.want)
		}
		// always two from the hole
		in := 0
		for _, c := range five {
			if slices.Contains(cards(tt.hole), c) {
				in++
			}
		}
		if in != 2 {
			t.Errorf("%s: %v uses %d hole cards", tt.name, five, in)
		}
	}
	plo5 := Game{Hole: 5}
	if _, score := plo5.Best(cards("2c 7d 9h As Ah"), cards("Ac Ad 5h 6s Kd")); HandName(score, nil) != "four of a kind" {
		t.Errorf("PLO5 picked %q", HandName(score, nil))
	}
	if c, score := plo.Best(cards("As Ad 2h 3c"), cards("Kc Kd")); c != nil || score <= worstScore {
		t.Error("scored a hand before the flop")
	}
}

func TestMadeHandOmaha(t *testing.T) {
	plo := Game{Hole: 4}
	if got := plo.MadeHand(cards("As Ad 2h 3c"), nil); got != "pair" {
		t.Errorf("preflop %q", got)
	}
	if got := plo.MadeHand(cards("As Kd Qc Jh"), cards("2s 5s 8s")); got != "high card" {
		t.Errorf("one suited card made %q", got)
	}
}
