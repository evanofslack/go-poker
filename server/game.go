package server

import (
	"fmt"

	"github.com/google/uuid"
)

type action string

type event struct {
	Action action         `json:"action"`
	Params map[string]any `json:"params"`
}

const actionSendMessage action = "send-message"
const actionNewMessage action = "new-message"
const actionNewPlayer action = "new-player"

const gameAdminName string = "system"

func processEvents(c *Client, e event) {
	switch e.Action {
	case actionSendMessage:
		handleSendMessage(c, e.Params["username"].(string), e.Params["message"].(string))
	case actionNewPlayer:
		handleNewPlayer(c, e.Params["username"].(string))
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
			"id":       uuid.New().String(),
			"message":  message,
			"username": username,
		},
	}
}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	c.hub.broadcast <- createNewMessageEvent(gameAdminName, fmt.Sprintf("%s has joined", username))
}
