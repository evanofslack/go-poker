package server

import (
	"fmt"
	"time"

	"github.com/alexclewontin/riverboat"
	"github.com/google/uuid"
)

type action string

type event struct {
	Action action         `json:"action"`
	Params map[string]any `json:"params"`
}

// inbound (client) actions
const actionSendMessage action = "send-message"
const actionNewPlayer action = "new-player"
const actionTakeSeat action = "take-seat"

// outbound (server) actions
const actionNewMessage action = "new-message"

const gameAdminName string = "system"

func processEvents(c *Client, e event) {
	switch e.Action {
	case actionSendMessage:
		handleSendMessage(c, e.Params["username"].(string), e.Params["message"].(string))
	case actionNewPlayer:
		handleNewPlayer(c, e.Params["username"].(string))
	case actionTakeSeat:
		handleTakeSeat(c, uint(e.Params["buyIn"].(float64))) // convert JSON/JS float to int
	default:
		fmt.Println("Unexpected Action")
	}
}

func handleSendMessage(c *Client, username string, message string) {
	c.hub.broadcast <- createNewMessageEvent(username, message)
}

func createNewMessageEvent(username string, message string) event {
	return event{
		Action: actionNewMessage,
		Params: map[string]any{
			"id":        uuid.New().String(),
			"message":   message,
			"username":  username,
			"timestamp": currentTime(),
		},
	}
}

func currentTime() string {
	return fmt.Sprintf("%d:%d", time.Now().Hour(), time.Now().Minute())
}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	c.hub.broadcast <- createNewMessageEvent(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, buyIn uint) {
	c.seatID = c.game.AddPlayer()
	err := riverboat.BuyIn(c.game, c.seatID, buyIn)
	if err != nil {
		fmt.Println(err)
	}
	err = riverboat.ToggleReady(c.game, c.seatID, 0)
	if err != nil {
		fmt.Println(err)
	}
	godView := c.game.GenerateOmniView()
	fmt.Printf("%+v\n", godView)
}
