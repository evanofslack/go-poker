package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/google/uuid"
)

const (
	gameAdminName string = "system"
	maxSeats      uint   = 6 // seats rendered by the frontend
)

func handleJoinTable(c *Client, tablename string) {
	table := c.hub.getOrCreateTable(tablename)
	if c.table != nil && c.table != table {
		c.table.unregister <- c
	}
	c.table = table
	table.register <- c
}

func handleLeaveTable(c *Client, tablename string) {
	table := c.hub.findTableByName(tablename)
	if table == nil {
		return
	}
	table.unregister <- c
	if c.table == table {
		c.table = nil
	}
}

func handleSendMessage(c *Client, username string, message string) {
	c.table.broadcast <- createNewMessage(username, message)
}

func handleSendLog(c *Client, message string) {
	c.table.broadcast <- createNewLog(message)
}

func handleNewPlayer(c *Client, username string) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	c.username = username
	c.send <- createUpdatedGame(c)
	c.table.broadcast <- createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, username string, seatID uint, buyIn uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	if seatID < 1 || seatID > maxSeats || buyIn == 0 {
		slog.Default().Warn("Take seat: invalid request", "seat", seatID, "buyIn", buyIn)
		return
	}
	for _, p := range c.table.game.GenerateOmniView().Players {
		if p.SeatID == seatID {
			slog.Default().Warn("Take seat: seat taken", "seat", seatID)
			return
		}
	}

	position := c.table.game.AddPlayer()
	c.uuid = c.table.game.GenerateOmniView().Players[position].UUID
	c.send <- createUpdatedPlayerUUID(c)
	err := poker.SetUsername(c.table.game, position, username)
	if err != nil {
		slog.Default().Warn("Set username", "error", err)
	}

	err = poker.BuyIn(c.table.game, position, buyIn)
	if err != nil {
		slog.Default().Warn("Buy in", "error", err)
	}

	// set player ready
	// TODO make this a separate action
	err = poker.ToggleReady(c.table.game, position, 0)
	if err != nil {
		slog.Default().Warn("Toggle ready", "error", err)
	}

	err = poker.SetSeatID(c.table.game, position, seatID)
	if err != nil {
		slog.Default().Warn("Set seat id", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleStartGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	err := c.table.game.Start()
	if err != nil {
		fmt.Println(err)
	}
	broadcastDeal(c.table)
	c.table.broadcast <- createUpdatedGame(c)
}

func handleResetGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	c.table.game.Reset()
	c.table.broadcast <- createUpdatedGame(c)
}

func handleDealGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	broadcastDeal(c.table)

	view := c.table.game.GenerateOmniView()
	err := poker.Deal(c.table.game, view.DealerNum, 0)
	if err != nil {
		slog.Default().Warn("Deal table", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleCall(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	if pn >= uint(len(view.Players)) {
		return
	}
	currentPlayer := view.Players[pn]

	// compute amount needed to call
	maxBet := view.Players[0].TotalBet
	for _, p := range view.Players {
		if p.TotalBet > maxBet {
			maxBet = p.TotalBet
		}
	}
	callAmount := maxBet - currentPlayer.TotalBet

	// if player must go all in to call
	if callAmount >= currentPlayer.Stack {
		callAmount = currentPlayer.Stack
	}

	err := poker.Bet(c.table.game, pn, callAmount)
	if err != nil {
		slog.Default().Warn("Handle call", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleRaise(c *Client, raise uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	if pn >= uint(len(view.Players)) {
		return
	}
	err := poker.Bet(c.table.game, pn, raise)
	if err != nil {
		slog.Default().Warn("Handle raise", "error", err)
	}

	c.table.broadcast <- createUpdatedGame(c)
}

func handleCheck(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	if pn >= uint(len(view.Players)) {
		return
	}
	err := poker.Bet(c.table.game, pn, 0)
	if err != nil {
		slog.Default().Warn("Handle check", "error", err)
	}
	c.table.broadcast <- createUpdatedGame(c)
}

func handleFold(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	view := c.table.game.GenerateOmniView()
	pn := view.ActionNum
	if pn >= uint(len(view.Players)) {
		return
	}
	err := poker.Fold(c.table.game, pn, 0)
	if err != nil {
		slog.Default().Warn("Handle fold", "error", err)
		return
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
		slog.Default().Warn("Marshal new message", "error", err)
	}
	return resp
}

func createNewLog(message string) []byte {
	log := newLog{
		base{actionNewLog},
		uuid.New().String(),
		message,
		currentTime(),
	}
	resp, err := json.Marshal(log)
	if err != nil {
		slog.Default().Warn("Marshal new log", "error", err)
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
		slog.Default().Warn("Marshal update game", "error", err)
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
		slog.Default().Warn("Marshal player uuid", "error", err)
	}
	return resp
}

func broadcastDeal(table *table) {
	view := table.game.GenerateOmniView()
	if view.SBNum >= uint(len(view.Players)) || view.BBNum >= uint(len(view.Players)) {
		return
	}

	startMsg := "starting new hand"
	table.broadcast <- createNewLog(startMsg)

	sbUser := view.Players[view.SBNum].Username
	sb := view.Config.SmallBlind
	sbMsg := fmt.Sprintf("%s is small blind (%d)", sbUser, sb)
	table.broadcast <- createNewLog(sbMsg)

	bbUser := view.Players[view.BBNum].Username
	bb := view.Config.BigBlind
	bbMsg := fmt.Sprintf("%s is big blind (%d)", bbUser, bb)
	table.broadcast <- createNewLog(bbMsg)
}

func currentTime() string {
	return fmt.Sprintf("%d:%02d", time.Now().Hour(), time.Now().Minute())
}
