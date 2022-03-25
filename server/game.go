package server

import "github.com/google/uuid"

type action string

type event struct {
	Action action                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

const actionSendMessage action = "send-message"

func processEvents(c *Client, e event) {
	if e.Action == actionSendMessage {
		handleSendMessage(c, e.Params["username"].(string), e.Params["message"].(string))
	}
}

func handleSendMessage(c *Client, username string, message string) {
	c.hub.broadcast <- createNewMessageEvent(username, message)
}

func createNewMessageEvent(username string, message string) event {
	return event{
		Action: "new-message",
		Params: map[string]interface{}{
			"id":       uuid.New().String(),
			"message":  message,
			"username": username,
		},
	}
}
