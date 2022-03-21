package main

type game struct {
	players []*player
	deck    *deck
}

// type gameService interface {
// 	deal()
// 	addPlayer(p *player) error
// 	removePlayer(p *player) error
// }

func newGame(players []*player) *game {
	deck := newDeck()
	return &game{
		players: players,
		deck:    deck,
	}
}

func (g *game) deal() {
	for i := 0; i < 2; i++ {
		for _, player := range g.players {
			player.hand[i] = g.deck.deal()

		}
	}
}
