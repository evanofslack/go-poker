package holdem

import (
	"strings"
	"testing"

	"github.com/alexclewontin/riverboat/eval"
)

func TestHandName(t *testing.T) {
	tests := map[string]string{
		"As Ks Qs Js Ts": "a royal flush",
		"9h 8h 7h 6h 5h": "a straight flush, nine high",
		"5d 4d 3d 2d Ad": "a straight flush, five high",
		"9s 9h 9d 9c 2s": "four of a kind, nines",
		"Js Jh Jd 4c 4s": "a full house, jacks full of fours",
		"6s 6h 6d Kc Ks": "a full house, sixes full of kings",
		"As 9s 7s 4s 2s": "a flush, ace high",
		"Ts 9h 8d 7c 6s": "a straight, ten high",
		"5s 4h 3d 2c Ah": "a straight, five high",
		"Qs Qh Qd 7c 2s": "three of a kind, queens",
		"Ks Kh 7d 7c 2s": "two pair, kings and sevens",
		"7s 7h Kd Kc 2s": "two pair, kings and sevens",
		"As Ah 9d 7c 2s": "a pair of aces",
		"Ah Jd 9c 7s 2h": "ace high",
	}
	for hand, want := range tests {
		var cards []eval.Card
		for _, s := range strings.Fields(hand) {
			cards = append(cards, eval.MustParseCardString(s))
		}
		score := eval.HandValue(cards[0], cards[1], cards[2], cards[3], cards[4])
		if got := HandName(score, cards); got != want {
			t.Errorf("HandName(%s) = %q, want %q", hand, got, want)
		}
	}

	// without cards, or without a showdown
	plain := map[int]string{166: "four of a kind", 3000: "two pair", 7462: "high card", 8000: "", 0: ""}
	for score, want := range plain {
		if got := HandName(score, nil); got != want {
			t.Errorf("HandName(%d, nil) = %q, want %q", score, got, want)
		}
	}
}
