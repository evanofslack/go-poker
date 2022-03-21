package main

type player struct {
	name  string
	hand  []*card
	chips int
}

func newPlayer(name string, chips int) *player {
	return &player{
		name:  name,
		hand:  make([]*card, 2),
		chips: chips,
	}
}
