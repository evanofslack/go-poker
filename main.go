package main

import "fmt"

func main() {
	player1 := newPlayer("test1", 1000)
	player2 := newPlayer("test2", 1000)

	players := []*player{player1, player2}

	game := newGame(players)
	game.deal()

	fmt.Println(game.players[0].name)
	fmt.Println(game.players[0].hand)
}
