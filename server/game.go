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
const sendMessage action = "send-message"
const newPlayer action = "new-player"
const takeSeat action = "take-seat"
const startGame action = "start-game"

// outbound (server) actions
const newMessage action = "new-message"
const updateGame action = "update-game"
const updateSeatID action = "update-seat"

const gameAdminName string = "system"

func processEvents(c *Client, e event) {
	switch e.Action {
	case sendMessage:
		handleSendMessage(c, e.Params["username"].(string), e.Params["message"].(string))
	case newPlayer:
		handleNewPlayer(c, e.Params["username"].(string))
	case takeSeat:
		handleTakeSeat(c, e.Params["username"].(string), uint(e.Params["position"].(float64)), uint(e.Params["buyIn"].(float64))) // convert JSON/JS float to int
	case startGame:
		handleStartGame(c)
	default:
		fmt.Println("Unexpected Action")
	}
}

func handleSendMessage(c *Client, username string, message string) {
	c.hub.broadcast <- createNewMessageEvent(username, message)
}

func handleTakeSeat(c *Client, username string, position uint, buyIn uint) {

	// add player to game
	c.seatID = c.game.AddPlayer()
	// set username
	err := riverboat.SetUsername(c.game, c.seatID, username)
	if err != nil {
		fmt.Println(err)
	}

	// player buy in
	err = riverboat.BuyIn(c.game, c.seatID, buyIn)
	if err != nil {
		fmt.Println(err)
	}

	// set player ready
	// TODO make this a separate action
	err = riverboat.ToggleReady(c.game, c.seatID, 0)
	if err != nil {
		fmt.Println(err)
	}

	// player position
	err = riverboat.SetPosition(c.game, c.seatID, position)
	if err != nil {
		fmt.Println(err)
	}

	// broadcast updated game
	event := createUpdatedGameEvent(c)
	c.hub.broadcast <- event
}

func handleStartGame(c *Client) {
	view := c.game.GenerateOmniView()
	err := riverboat.Deal(c.game, view.DealerNum, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func createNewMessageEvent(username string, message string) event {
	return event{
		Action: newMessage,
		Params: map[string]any{
			"id":        uuid.New().String(),
			"message":   message,
			"username":  username,
			"timestamp": currentTime(),
		},
	}
}

func createUpdatedGameEvent(c *Client) event {
	return event{
		Action: updateGame,
		Params: gameSerializer(c.game),
	}
}

func createUpdatedSeatIDEvent(seatID uint) event {
	return event{
		Action: updateSeatID,
		Params: map[string]any{
			"seatID": seatID,
		},
	}
}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	c.send <- createUpdatedGameEvent(c)
	c.hub.broadcast <- createNewMessageEvent(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func gameSerializer(g *riverboat.Game) map[string]any {
	view := g.GenerateOmniView()

	config := map[string]any{
		"maxBuyIn":   view.Config.MaxBuy,
		"bigBlind":   view.Config.BigBlind,
		"smallBlind": view.Config.SmallBlind,
	}

	players := make([]map[string]interface{}, 0)
	for _, p := range view.Players {
		player := map[string]any{
			"username":   p.Username,
			"position":   p.Position,
			"ready":      p.Ready,
			"in":         p.In,
			"called":     p.Called,
			"left":       p.Left,
			"totalBuyIn": p.TotalBuyIn,
			"stack":      p.Stack,
			"bet":        p.Bet,
			"totalBet":   p.TotalBet,
			"cards":      p.Cards,
		}
		players = append(players, player)
	}

	pots := make([]map[string]interface{}, 0)
	for _, p := range view.Pots {
		pot := map[string]any{
			"topShare":           p.TopShare,
			"amt":                p.Amt,
			"eligiblePlayerNums": p.EligiblePlayerNums,
			"winningPlayerNums":  p.WinningPlayerNums,
			"winningHand":        p.WinningHand,
			"winningScore":       p.WinningScore,
		}
		pots = append(pots, pot)
	}

	params := map[string]any{
		"dealer":         view.DealerNum,
		"action":         view.ActionNum,
		"utg":            view.UTGNum,
		"sb":             view.SBNum,
		"bb":             view.BBNum,
		"communityCards": view.CommunityCards,
		"stage":          view.Stage,
		"betting":        view.Betting,
		"config":         config,
		"players":        players,
		"pots":           pots,
		"minRaise":       view.MinRaise,
		"readyCount":     view.ReadyCount,
	}
	return params
}

func currentTime() string {
	return fmt.Sprintf("%d:%02d", time.Now().Hour(), time.Now().Minute())
}
