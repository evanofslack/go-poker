package server

import (
	"fmt"
	"time"

	"github.com/alexclewontin/riverboat"
	"github.com/alexclewontin/riverboat/eval"
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
const dealGame action = "deal-game"
const playerCall action = "player-call"
const playerCheck action = "player-check"
const playerRaise action = "player-raise"
const playerFold action = "player-fold"

// outbound (server) actions
const newMessage action = "new-message"
const updateGame action = "update-game"
const updatePlayerID action = "update-player-id"

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
	case dealGame:
		handleDealGame(c)
	case playerCall:
		handleCall(c)
	case playerCheck:
		handleCheck(c)
	case playerRaise:
		handleRaise(c, uint(e.Params["amount"].(float64)))
	case playerFold:
		handleFold(c)
	default:
		fmt.Println("Unexpected Action")
	}
}

func handleSendMessage(c *Client, username string, message string) {
	c.hub.broadcast <- createNewMessageEvent(username, message)

}

func handleNewPlayer(c *Client, username string) {
	c.username = username
	c.send <- createUpdatedGameEvent(c)
	c.hub.broadcast <- createNewMessageEvent(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, username string, position uint, buyIn uint) {

	seatID := c.game.AddPlayer()
	c.id = c.game.GenerateOmniView().Players[seatID].ID
	c.send <- createUpdatedPlayerIDEvent(c)
	err := riverboat.SetUsername(c.game, seatID, username)
	if err != nil {
		fmt.Println(err)
	}

	err = riverboat.BuyIn(c.game, seatID, buyIn)
	if err != nil {
		fmt.Println(err)
	}

	// set player ready
	// TODO make this a separate action
	err = riverboat.ToggleReady(c.game, seatID, 0)
	if err != nil {
		fmt.Println(err)
	}

	err = riverboat.SetPosition(c.game, seatID, position)
	if err != nil {
		fmt.Println(err)
	}

	event := createUpdatedGameEvent(c) // broadcast updated game
	c.hub.broadcast <- event
}

func handleStartGame(c *Client) {
	err := c.game.Start()
	if err != nil {
		fmt.Println(err)
	}
	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func handleDealGame(c *Client) {
	view := c.game.GenerateOmniView()
	err := riverboat.Deal(c.game, view.DealerNum, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func handleCall(c *Client) {
	view := c.game.GenerateOmniView()
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

	err := riverboat.Bet(c.game, pn, callAmount)
	if err != nil {
		fmt.Println(err)
	}
	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func handleRaise(c *Client, raise uint) {
	view := c.game.GenerateOmniView()
	pn := view.ActionNum
	err := riverboat.Bet(c.game, pn, raise)
	if err != nil {
		fmt.Println(err)
	}

	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func handleCheck(c *Client) {
	view := c.game.GenerateOmniView()
	pn := view.ActionNum
	err := riverboat.Bet(c.game, pn, 0)
	if err != nil {
		fmt.Println(err)
	}
	c.hub.broadcast <- createUpdatedGameEvent(c)
}

func handleFold(c *Client) {
	view := c.game.GenerateOmniView()
	pn := view.ActionNum
	err := riverboat.Fold(c.game, pn, 0)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(c.username, "folds")
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

func createUpdatedPlayerIDEvent(c *Client) event {
	return event{
		Action: updatePlayerID,
		Params: map[string]any{"id": c.id},
	}
}

func cardReader(cards []eval.Card) []string {
	var readable []string
	for _, c := range cards {
		readable = append(readable, c.String())
	}
	return readable
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
			"id":         p.ID,
			"position":   p.Position,
			"seatID":     p.SeatID,
			"ready":      p.Ready,
			"in":         p.In,
			"called":     p.Called,
			"left":       p.Left,
			"totalBuyIn": p.TotalBuyIn,
			"stack":      p.Stack,
			"bet":        p.Bet,
			"totalBet":   p.TotalBet,
			"cards":      cardReader(p.Cards[:]),
		}
		players = append(players, player)
	}

	pots := make([]map[string]any, 0)
	for _, p := range view.Pots {
		pot := map[string]any{
			"topShare":           p.TopShare,
			"amount":             p.Amt,
			"eligiblePlayerNums": p.EligiblePlayerNums,
			"winningPlayerNums":  p.WinningPlayerNums,
			"winningHand":        p.WinningHand,
			"winningScore":       p.WinningScore,
		}
		pots = append(pots, pot)
	}

	params := map[string]any{
		"running":        view.Running,
		"dealer":         view.DealerNum,
		"action":         view.ActionNum,
		"utg":            view.UTGNum,
		"sb":             view.SBNum,
		"bb":             view.BBNum,
		"communityCards": cardReader(view.CommunityCards),
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
