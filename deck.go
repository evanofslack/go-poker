package main

type deck struct {
	cards []*card
}

// type deckService interface {
// 	shuffle()
// 	deal() *card
// }

func (d *deck) shuffle() {
	for i := 0; i < len(d.cards); i++ {
		//shuffle
	}

}

func (d *deck) deal() *card {
	card := d.cards[len(d.cards)-1]

	return card
}

func newDeck() *deck {

	deck := new(deck)
	suits := []suit{"hearts", "diamonds", "spades", "clubs"}
	values := []denomination{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

	for _, suit := range suits {
		for _, value := range values {
			card := newCard(suit, value)
			deck.cards = append(deck.cards, card)
		}
	}

	deck.shuffle()

	return deck
}
