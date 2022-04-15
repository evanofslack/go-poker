package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/google/uuid"
)

const gameAdminName string = "system"

func handleJoinTable(c *Client, tablename string) {
	table := c.hub.findTableByName(tablename)
	if table == nil {
		table = c.hub.createTable(tablename)
	}
	c.table = table
	table.register <- c
}

func handleLeaveTable(c *Client, tablename string) {
	table := c.hub.findTableByName(tablename)
	table.unregister <- c
}

func handleSendMessage(c *Client, username string, message string) {
	c.table.broadcast <- createNewMessage(username, message)

}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	c.send <- createUpdatedGame(c)
	c.table.broadcast <- createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, username string, seatID uint, buyIn uint) {

	position := c.table.game.AddPlayer()
	c.uuid = c.table.game.GenerateOmniView().Players[position].UUID
	c.send <- createUpdatedPlayerUUID(c)
	err := poker.SetUsername(c.table.game, position, username)
	if err != nil {
		fmt.Println(err)
	}

	err = poker.BuyIn(c.table.game, position, buyIn)
	if err != nil {
		fmt.Println(err)
	}

	// set player ready
	// TODO make this a separate action
	err = poker.ToggleReady(c.table.game, position, 0)
	if err != nil {
		fmt.Println(err)
	}

	err = poker.SetSeatID(c.table.game, position, seatID)
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleStartGame(c *Client) {
	err := c.table.game.Start()
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleResetGame(c *Client) {
	c.table.game.Reset()
	c.table.broadcast <- createUpdatedGame(c)
}

func handleDealGame(c *Client) {
	view := c.table.game.GenerateOmniView()
	err := poker.Deal(c.table.game, view.DealerNum, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleCall(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	current_player := view.Players[pn]

	// compute amount needed to call
	maxBet := view.Players[0].TotalBet
	for _, p := range view.Players {
		if p.TotalBet > maxBet {
			maxBet = p.TotalBet
		}
	}
	callAmount := maxBet - current_player.TotalBet

	// player must go all in to call
	if callAmount >= current_player.Stack {
		callAmount = current_player.Stack
	}

	err := poker.Bet(c.table.game, pn, callAmount)
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleRaise(c *Client, raise uint) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Bet(c.table.game, pn, raise)
	if err != nil {
		fmt.Println(err)
	}

	c.table.broadcast <- createUpdatedGame(c)
}

func handleCheck(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Bet(c.table.game, pn, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleFold(c *Client) {
	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	err := poker.Fold(c.table.game, pn, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func createNewMessage(username string, message string) []byte {
	new := newMessage{
		base{actionNewMessage},
		uuid.New().String(),
		message,
		username,
		currentTime(),
	}
	resp, err := json.Marshal(new)
	if err != nil {
		fmt.Println(err)
	}
	return resp
}

func createUpdatedGame(c *Client) []byte {
	game := updateGame{
		base{actionUpdateGame},
		c.table.game.GenerateOmniView(),
	}

	resp, err := json.Marshal(game)
	if err != nil {
		fmt.Println(err)
	}
	return resp
}

func createUpdatedPlayerUUID(c *Client) []byte {
	uuid := updatePlayerUUID{
		base{actionUpdatePlayerUUID},
		c.uuid,
	}
	resp, err := json.Marshal(uuid)
	if err != nil {
		fmt.Println(err)
	}
	return resp
}

func currentTime() string {
	return fmt.Sprintf("%d:%02d", time.Now().Hour(), time.Now().Minute())
}
