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

func handleJoinTable(c *Client, tablename string) error {
	return c.hub.joinTable(c, tablename)
}

func handleLeaveTable(c *Client, tablename string) {
	if c.table != nil && c.table.name == tablename {
		c.hub.leaveTable(c)
	}
}

func handleSendMessage(c *Client, message string) {
	// use the name the server knows, not one from the message
	if c.username == "" {
		return
	}
	c.table.broadcast <- createNewMessage(c.username, message)
}

func handleSendLog(c *Client, message string) {
	c.table.broadcast <- createNewLog(message)
}

func handleNewPlayer(c *Client, username string) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	c.username = username
	c.send <- createGameView(c)
	c.table.broadcast <- createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", username))
}

func handleTakeSeat(c *Client, username string, seatID uint, buyIn uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	maxBuy := c.table.game.GenerateOmniView().Config.MaxBuy
	if seatID < 1 || seatID > maxSeats || buyIn == 0 || buyIn > maxBuy {
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
	c.uuid.Store(c.table.game.GenerateOmniView().Players[position].UUID)
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
	broadcastGame(c.table)
}

func handleStartGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if !isSeated(c) {
		return
	}

	err := c.table.game.Start()
	if err != nil {
		fmt.Println(err)
	}
	broadcastDeal(c.table)
	broadcastGame(c.table)
}

func handleResetGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if !isSeated(c) {
		return
	}

	c.table.game.Reset()
	broadcastGame(c.table)
}

func handleDealGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if !isSeated(c) {
		return
	}

	broadcastDeal(c.table)

	view := c.table.game.GenerateOmniView()
	err := poker.Deal(c.table.game, view.DealerNum, 0)
	if err != nil {
		slog.Default().Warn("Deal table", "error", err)
	}
	broadcastGame(c.table)
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
	broadcastGame(c.table)
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

	broadcastGame(c.table)
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
	broadcastGame(c.table)
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
	broadcastGame(c.table)
}

// playerNum finds c's player index. Indexes change as players sit down, so
// look it up for every action. Caller holds c.table.mu.
func playerNum(c *Client) (uint, bool) {
	for i, p := range c.table.game.GenerateOmniView().Players {
		if p.UUID == c.id() {
			return uint(i), true
		}
	}
	return 0, false
}

// isSeated reports whether c has a seat at its table. Caller holds c.table.mu.
func isSeated(c *Client) bool {
	_, ok := playerNum(c)
	return ok
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

// broadcastGame sends every client its own view of the game. Caller holds t.mu.
func broadcastGame(t *table) {
	u := stateUpdate{byUUID: map[string][]byte{}}
	players := t.game.GenerateOmniView().Players
	for i, p := range players {
		u.byUUID[p.UUID] = createUpdatedGame(t.game.GeneratePlayerView(uint(i)))
	}
	u.spectator = createUpdatedGame(t.game.GeneratePlayerView(uint(len(players))))
	t.updates <- u
}

// createGameView is c's own view of the game. Caller holds c.table.mu.
func createGameView(c *Client) []byte {
	pn, ok := playerNum(c)
	if !ok {
		pn = uint(len(c.table.game.GenerateOmniView().Players))
	}
	return createUpdatedGame(c.table.game.GeneratePlayerView(pn))
}

func createUpdatedGame(view *poker.GameView) []byte {
	game := updateGame{
		base{actionUpdateGame},
		view,
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
		c.id(),
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
