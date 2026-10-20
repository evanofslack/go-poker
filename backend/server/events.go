package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/google/uuid"
)

const (
	gameAdminName string = "system"
	maxSeats      uint   = 6 // seats rendered by the frontend
)

// Vars so tests can shorten them.
var (
	nextHandDelay = 5 * time.Second  // pause after a hand before dealing the next
	turnTimeout   = 30 * time.Second // time to act before checking or folding for you
)

func handleJoinTable(c *Client, tablename string, username string, create bool) {
	if err := c.hub.joinTable(c, tablename, username, create); err != nil {
		sendError(c, err.Error())
		return
	}
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	c.send <- createJoinedTable(c)
	c.send <- createGameView(c)
	c.table.send(createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", username)))
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
	c.table.send(createNewMessage(c.username, message))
}

func handleTakeSeat(c *Client, seatID uint, buyIn uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	if isSeated(c) {
		sendError(c, "you already have a seat")
		return
	}
	maxBuy := c.table.game.GenerateOmniView().Config.MaxBuy
	if seatID < 1 || seatID > maxSeats {
		sendError(c, "invalid seat")
		return
	}
	if buyIn == 0 || buyIn > maxBuy {
		sendError(c, fmt.Sprintf("buy-in must be between 1 and %d", maxBuy))
		return
	}
	for _, p := range c.table.game.GenerateOmniView().Players {
		if p.SeatID == seatID {
			sendError(c, "seat taken")
			return
		}
	}

	position := c.table.game.AddPlayer()
	c.uuid.Store(c.table.game.GenerateOmniView().Players[position].UUID)
	c.send <- createUpdatedPlayerUUID(c)
	err := poker.SetUsername(c.table.game, position, c.username)
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
	c.table.send(createNewLog(fmt.Sprintf("%s sits down with %d", c.username, buyIn)))

	// a running game may be waiting for players
	view := c.table.game.GenerateOmniView()
	if view.Running && view.Stage == poker.PreDeal {
		c.table.scheduleDeal()
	}
	broadcastGame(c.table)
}

func handleStartGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if !isSeated(c) {
		sendError(c, "take a seat first")
		return
	}

	err := c.table.game.Start()
	if err != nil {
		sendError(c, "need at least two ready players")
		return
	}
	logDeal(c.table)
	c.table.startTurnTimer()
	broadcastGame(c.table)
}

func handleResetGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if c.table.host != c {
		sendError(c, "only the host can reset the table")
		return
	}

	if c.table.dealTimer != nil {
		c.table.dealTimer.Stop()
	}
	c.table.turnSeq++ // cancel the turn timer
	c.table.game.Reset()
	c.table.send(createNewLog(fmt.Sprintf("%s reset the table", c.username)))
	broadcastGame(c.table)
}

func handleCall(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	view := c.table.game.GenerateOmniView()
	p := view.Players[pn]

	// call this street's top bet, or go all in if that's more than the stack
	var maxBet uint
	for _, q := range view.Players {
		maxBet = max(maxBet, q.Bet)
	}
	callAmount := min(maxBet-p.Bet, p.Stack)

	if err := poker.Bet(c.table.game, pn, callAmount); err != nil {
		sendError(c, "can't call")
		return
	}
	finishAction(c.table, fmt.Sprintf("%s calls %d", p.Username, callAmount))
}

func handleRaise(c *Client, raise uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	view := c.table.game.GenerateOmniView()
	p := view.Players[pn]
	var maxBet uint
	for _, q := range view.Players {
		maxBet = max(maxBet, q.Bet)
	}

	if err := poker.Bet(c.table.game, pn, raise); err != nil {
		sendError(c, "invalid bet")
		return
	}
	msg := fmt.Sprintf("%s bets %d", p.Username, raise)
	if maxBet > 0 {
		msg = fmt.Sprintf("%s raises to %d", p.Username, p.Bet+raise)
	}
	finishAction(c.table, msg)
}

func handleCheck(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	if err := poker.Bet(c.table.game, pn, 0); err != nil {
		sendError(c, "can't check")
		return
	}
	finishAction(c.table, fmt.Sprintf("%s checks", c.username))
}

func handleFold(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	if err := poker.Fold(c.table.game, pn, 0); err != nil {
		sendError(c, "can't fold")
		return
	}
	finishAction(c.table, fmt.Sprintf("%s folds", c.username))
}

// finishAction logs a successful action and sends the new game state. If the
// action ended the hand it also logs the winners and deals the next hand
// after a pause. Caller holds t.mu.
func finishAction(t *table, msg string) {
	if msg != "" {
		t.send(createNewLog(msg))
	}
	foldLeavers(t)
	view := t.game.GenerateOmniView()
	if view.Stage == poker.PreDeal {
		logWinners(t, view)
		removeLeavers(t)
		t.scheduleDeal()
	} else {
		t.startTurnTimer()
	}
	broadcastGame(t)
}

// scheduleDeal deals the next hand after a pause. Caller holds t.mu.
func (t *table) scheduleDeal() {
	if t.dealTimer != nil {
		t.dealTimer.Stop()
	}
	t.dealTimer = time.AfterFunc(nextHandDelay, func() { dealNextHand(t) })
}

// startTurnTimer gives the player to act turnTimeout to do so. Caller holds t.mu.
func (t *table) startTurnTimer() {
	t.turnSeq++
	seq := t.turnSeq
	if t.turnTimer != nil {
		t.turnTimer.Stop()
	}
	t.turnTimer = time.AfterFunc(turnTimeout, func() { turnTimedOut(t, seq) })
}

// turnTimedOut checks for the player to act if they can, and folds otherwise.
func turnTimedOut(t *table, seq int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case <-t.done:
		return
	default:
	}
	view := t.game.GenerateOmniView()
	if seq != t.turnSeq || !view.Betting {
		return
	}
	pn := view.ActionNum
	p := view.Players[pn]
	var maxBet uint
	for _, q := range view.Players {
		maxBet = max(maxBet, q.Bet)
	}
	if p.Bet >= maxBet && poker.Bet(t.game, pn, 0) == nil {
		finishAction(t, fmt.Sprintf("%s checks (out of time)", p.Username))
		return
	}
	if err := poker.Fold(t.game, pn, 0); err != nil {
		slog.Default().Warn("Turn timeout fold", "error", err)
		return
	}
	finishAction(t, fmt.Sprintf("%s folds (out of time)", p.Username))
}

// leaveSeat takes the player with uuid out of the game: right away between
// hands, otherwise by folding them when it's their turn. Caller holds t.mu.
func leaveSeat(t *table, uuid string) {
	view := t.game.GenerateOmniView()
	seated := false
	for _, p := range view.Players {
		seated = seated || p.UUID == uuid
	}
	if !seated {
		return
	}
	t.leaving[uuid] = true
	switch {
	case !view.Betting && view.Stage == poker.PreDeal:
		removeLeavers(t)
		broadcastGame(t)
	case view.Betting && view.Players[view.ActionNum].UUID == uuid:
		finishAction(t, "") // folds them
	}
}

// foldLeavers folds players who left once action reaches them. Caller holds t.mu.
func foldLeavers(t *table) {
	for {
		view := t.game.GenerateOmniView()
		if !view.Betting {
			return
		}
		p := view.Players[view.ActionNum]
		if !t.leaving[p.UUID] {
			return
		}
		if err := poker.Fold(t.game, view.ActionNum, 0); err != nil {
			slog.Default().Warn("Fold leaver", "error", err)
			return
		}
		t.send(createNewLog(fmt.Sprintf("%s folds", p.Username)))
	}
}

// removeLeavers removes players who left from the game. Only works between
// hands. Caller holds t.mu.
func removeLeavers(t *table) {
	for uuid := range t.leaving {
		for pn, p := range t.game.GenerateOmniView().Players {
			if p.UUID != uuid {
				continue
			}
			if err := poker.RemovePlayer(t.game, uint(pn), 0); err != nil {
				return // mid-hand, try again after it
			}
			t.send(createNewLog(fmt.Sprintf("%s left the table", p.Username)))
		}
		delete(t.leaving, uuid)
	}
}

// dealNextHand deals a new hand, unless the table was reset or deleted.
func dealNextHand(t *table) {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case <-t.done:
		return
	default:
	}
	view := t.game.GenerateOmniView()
	if !view.Running || view.Stage != poker.PreDeal {
		return
	}
	if err := poker.Deal(t.game, view.DealerNum, 0); err != nil {
		t.send(createNewLog("waiting for players"))
		return
	}
	logDeal(t)
	t.startTurnTimer()
	broadcastGame(t)
}

func logWinners(t *table, view *poker.GameView) {
	for _, pot := range view.Pots {
		if pot.Amt == 0 || len(pot.WinningPlayerNums) == 0 {
			continue
		}
		var names []string
		for _, pn := range pot.WinningPlayerNums {
			names = append(names, view.Players[pn].Username)
		}
		msg := fmt.Sprintf("%s wins %d", strings.Join(names, " and "), pot.Amt)
		if len(names) > 1 {
			msg = fmt.Sprintf("%s split %d", strings.Join(names, " and "), pot.Amt)
		}
		if hand := handName(pot.WinningScore); hand != "" {
			msg += " with " + hand
		}
		t.send(createNewLog(msg))
	}
}

// handName names a hand from its score, where lower is better (Cactus Kev's
// ranks). A hand won without a showdown scores 8000 and has no name.
func handName(score int) string {
	switch {
	case score <= 0 || score > 7462:
		return ""
	case score <= 10:
		return "a straight flush"
	case score <= 166:
		return "four of a kind"
	case score <= 322:
		return "a full house"
	case score <= 1599:
		return "a flush"
	case score <= 1609:
		return "a straight"
	case score <= 2467:
		return "three of a kind"
	case score <= 3325:
		return "two pair"
	case score <= 6185:
		return "a pair"
	default:
		return "high card"
	}
}

// actingPlayer returns c's player index if it is c's turn, and otherwise
// tells c why not. Caller holds c.table.mu.
func actingPlayer(c *Client) (uint, bool) {
	pn, ok := playerNum(c)
	if !ok {
		sendError(c, "take a seat first")
		return 0, false
	}
	view := c.table.game.GenerateOmniView()
	if !view.Betting || view.ActionNum != pn {
		sendError(c, "not your turn")
		return 0, false
	}
	return pn, true
}

// sendError tells c its action was rejected. It never blocks, so it is safe
// while holding c.table.mu.
func sendError(c *Client, message string) {
	resp, err := json.Marshal(errorMessage{base{actionError}, message, currentTime()})
	if err != nil {
		slog.Default().Warn("Marshal error", "error", err)
		return
	}
	select {
	case c.send <- resp:
	default:
	}
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
	host := t.hostID()
	for i, p := range players {
		u.byUUID[p.UUID] = createUpdatedGame(t.game.GeneratePlayerView(uint(i)), host)
	}
	u.spectator = createUpdatedGame(t.game.GeneratePlayerView(uint(len(players))), host)
	select {
	case t.updates <- u:
	case <-t.done:
	}
}

// createGameView is c's own view of the game. Caller holds c.table.mu.
func createGameView(c *Client) []byte {
	pn, ok := playerNum(c)
	if !ok {
		pn = uint(len(c.table.game.GenerateOmniView().Players))
	}
	return createUpdatedGame(c.table.game.GeneratePlayerView(pn), c.table.hostID())
}

func createUpdatedGame(view *poker.GameView, host string) []byte {
	game := updateGame{
		base{actionUpdateGame},
		view,
		host,
	}

	resp, err := json.Marshal(game)
	if err != nil {
		slog.Default().Warn("Marshal update game", "error", err)
	}
	return resp
}

func createJoinedTable(c *Client) []byte {
	joined := joinedTable{
		base{actionJoinedTable},
		c.table.name,
		c.username,
		c.id(),
	}
	resp, err := json.Marshal(joined)
	if err != nil {
		slog.Default().Warn("Marshal joined table", "error", err)
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

// logDeal logs a new hand and its blinds. Caller holds t.mu.
func logDeal(t *table) {
	view := t.game.GenerateOmniView()
	if view.SBNum >= uint(len(view.Players)) || view.BBNum >= uint(len(view.Players)) {
		return
	}
	t.send(createNewLog("starting new hand"))
	t.send(createNewLog(fmt.Sprintf("%s is small blind (%d)", view.Players[view.SBNum].Username, view.Config.SmallBlind)))
	t.send(createNewLog(fmt.Sprintf("%s is big blind (%d)", view.Players[view.BBNum].Username, view.Config.BigBlind)))
}

func currentTime() string {
	return fmt.Sprintf("%d:%02d", time.Now().Hour(), time.Now().Minute())
}
