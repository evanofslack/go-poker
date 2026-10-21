package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/google/uuid"
)

const gameAdminName string = "system"

// how long a disconnected player keeps their seat, a var so tests can shorten it
var reconnectGrace = 60 * time.Second

func handleJoinTable(c *Client, tablename, username, token, password string, create bool, settings tableSettings) {
	reconnected, err := c.hub.joinTable(c, tablename, username, token, password, create, settings)
	if err != nil {
		sendErrorCode(c, errorCodes[err], err.Error())
		return
	}
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	c.send <- createJoinedTable(c)
	c.send <- createGameView(c)
	broadcastGame(c.table) // the watching count may have changed
	switch {
	case reconnected: // already logged
	case c.table.inLedger(c.username):
		c.table.send(createNewMessage(gameAdminName, fmt.Sprintf("%s rejoined, continuing their ledger", c.username)))
	default:
		c.table.send(createNewMessage(gameAdminName, fmt.Sprintf("%s has joined", c.username)))
	}
}

func handleLeaveTable(c *Client, tablename string) {
	if c.table != nil && c.table.name == tablename {
		c.hub.leaveTable(c, false)
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
	if err := checkSeat(c.table, seatID, buyIn); err != nil {
		sendError(c, err.Error())
		return
	}
	uuid := seatPlayer(c.table, c.username, seatID, buyIn)
	c.uuid.Store(uuid)
	token := newUUID()
	c.table.tokens[token] = uuid
	c.send <- createUpdatedPlayerUUID(c, token)
	broadcastGame(c.table)
}

// checkSeat returns why a new player can't sit in seatID with buyIn, if they
// can't. Caller holds t.mu.
func checkSeat(t *table, seatID uint, buyIn uint) error {
	s := t.settings
	if seatID < 1 || seatID > s.Seats {
		return errors.New("invalid seat")
	}
	if buyIn < s.MinBuy || buyIn > s.MaxBuy {
		return fmt.Errorf("buy-in must be between %d and %d", s.MinBuy, s.MaxBuy)
	}
	for _, p := range t.game.GenerateOmniView().Players {
		if p.SeatID == seatID {
			return errors.New("seat taken")
		}
	}
	return nil
}

// seatPlayer sits a new player down, ready to play, and returns their uuid.
// Caller holds t.mu and has checked the seat with checkSeat.
func seatPlayer(t *table, name string, seatID uint, buyIn uint) string {
	position := t.game.AddPlayer()
	uuid := t.game.GenerateOmniView().Players[position].UUID
	err := poker.SetUsername(t.game, position, name)
	if err != nil {
		slog.Default().Warn("Set username", "error", err)
	}

	err = poker.BuyIn(t.game, position, buyIn)
	if err != nil {
		slog.Default().Warn("Buy in", "error", err)
	}

	// set player ready
	// TODO make this a separate action
	err = poker.ToggleReady(t.game, position, 0)
	if err != nil {
		slog.Default().Warn("Toggle ready", "error", err)
	}

	err = poker.SetSeatID(t.game, position, seatID)
	if err != nil {
		slog.Default().Warn("Set seat id", "error", err)
	}
	t.record(name, entryBuyIn, buyIn, name)
	t.send(createNewLog(fmt.Sprintf("%s sits down with %d", name, buyIn)))

	// a running game may be waiting for players
	view := t.game.GenerateOmniView()
	if view.Running && view.Stage == poker.PreDeal {
		t.scheduleDeal()
	}
	return uuid
}

// handleStartGame starts the game. Any seated player can start it, and so can
// the host without a seat, for a game of only bots.
func handleStartGame(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	if !isSeated(c) && c.table.host != c {
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
	// everyone leaves with their chips, including any bet this hand
	for _, p := range c.table.game.GenerateOmniView().Players {
		c.table.record(p.Username, entryCashOut, p.Stack+p.TotalBet, c.username)
	}
	c.table.game.Reset()
	clear(c.table.sitOutNext)
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
	if call(c.table, pn) != nil {
		sendError(c, "can't call")
	}
}

func handleRaise(c *Client, amount uint) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	if raise(c.table, pn, amount) != nil {
		sendError(c, "invalid bet")
	}
}

func handleCheck(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	if check(c.table, pn) != nil {
		sendError(c, "can't check")
	}
}

func handleFold(c *Client) {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()

	pn, ok := actingPlayer(c)
	if !ok {
		return
	}
	if fold(c.table, pn) != nil {
		sendError(c, "can't fold")
	}
}

// call has pn call this street's top bet, or go all in if that's more than
// their stack. Caller holds t.mu, and has checked it's pn's turn.
func call(t *table, pn uint) error {
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	var maxBet uint
	for _, q := range view.Players {
		maxBet = max(maxBet, q.Bet)
	}
	callAmount := min(maxBet-p.Bet, p.Stack)

	if err := poker.Bet(t.game, pn, callAmount); err != nil {
		return err
	}
	finishAction(t, fmt.Sprintf("%s calls %d", p.Username, callAmount))
	return nil
}

// raise has pn put in amount more chips, as a bet or raise. Caller holds t.mu.
func raise(t *table, pn uint, amount uint) error {
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	var maxBet uint
	for _, q := range view.Players {
		maxBet = max(maxBet, q.Bet)
	}

	if err := poker.Bet(t.game, pn, amount); err != nil {
		return err
	}
	msg := fmt.Sprintf("%s bets %d", p.Username, amount)
	if maxBet > 0 {
		msg = fmt.Sprintf("%s raises to %d", p.Username, p.Bet+amount)
	}
	finishAction(t, msg)
	return nil
}

// check has pn check. Caller holds t.mu.
func check(t *table, pn uint) error {
	name := t.game.GenerateOmniView().Players[pn].Username
	if err := poker.Bet(t.game, pn, 0); err != nil {
		return err
	}
	finishAction(t, fmt.Sprintf("%s checks", name))
	return nil
}

// fold has pn fold. Caller holds t.mu.
func fold(t *table, pn uint) error {
	name := t.game.GenerateOmniView().Players[pn].Username
	if err := poker.Fold(t.game, pn, 0); err != nil {
		return err
	}
	finishAction(t, fmt.Sprintf("%s folds", name))
	return nil
}

// handleLeaveSeat gives up c's seat but keeps c at the table as a spectator.
// Mid-hand the player is folded on their turn and removed after the hand.
func handleLeaveSeat(c *Client) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	if !isSeated(c) {
		sendError(c, "you don't have a seat")
		return
	}
	old := c.id()
	leaveSeat(t, old)
	for token, u := range t.tokens {
		if u == old {
			delete(t.tokens, token)
		}
	}
	// a fresh uuid makes c a spectator, even while its old player finishes the hand
	c.uuid.Store(newUUID())
	c.send <- createUpdatedPlayerUUID(c, "")
	broadcastGame(t)
}

// handleSitOut toggles c sitting out. Between hands it takes effect right
// away, mid-hand once the hand ends.
func handleSitOut(c *Client) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	pn, ok := playerNum(c)
	if !ok {
		sendError(c, "take a seat first")
		return
	}
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	switch {
	case t.sitOutNext[p.UUID]:
		delete(t.sitOutNext, p.UUID)
		t.send(createNewLog(fmt.Sprintf("%s will play the next hand", p.Username)))
	case p.Ready && view.Betting:
		// toggling mid-hand could move the dealer, so wait for the hand to end
		t.sitOutNext[p.UUID] = true
		t.send(createNewLog(fmt.Sprintf("%s will sit out the next hand", p.Username)))
	case p.Ready:
		if err := poker.ToggleReady(t.game, pn, 0); err != nil {
			sendError(c, "can't sit out right now")
			return
		}
		t.send(createNewLog(fmt.Sprintf("%s sits out", p.Username)))
	case p.Stack == 0:
		sendError(c, "you're out of chips, rebuy first")
		return
	default:
		if err := poker.ToggleReady(t.game, pn, 0); err != nil {
			sendError(c, "can't sit back in right now")
			return
		}
		t.send(createNewLog(fmt.Sprintf("%s is back", p.Username)))
		if view.Running && view.Stage == poker.PreDeal {
			t.scheduleDeal()
		}
	}
	broadcastGame(t)
}

// applySitOuts sits out players who asked to during the last hand. Only
// works between hands. Caller holds t.mu.
func applySitOuts(t *table) {
	for uuid := range t.sitOutNext {
		delete(t.sitOutNext, uuid)
		for pn, p := range t.game.GenerateOmniView().Players {
			if p.UUID != uuid || !p.Ready {
				continue // busted players are already out
			}
			if err := poker.ToggleReady(t.game, uint(pn), 0); err != nil {
				slog.Default().Warn("Sit out", "error", err)
				continue
			}
			t.send(createNewLog(fmt.Sprintf("%s sits out", p.Username)))
		}
	}
}

// handleRebuy adds amount to c's stack while c isn't in a hand, and deals a
// busted player back in.
func handleRebuy(c *Client, amount uint) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	pn, ok := playerNum(c)
	if !ok {
		sendError(c, "take a seat first")
		return
	}
	view := t.game.GenerateOmniView()
	p := view.Players[pn]
	if p.In {
		sendError(c, "can't rebuy during a hand")
		return
	}
	// topping up can be any amount, but a busted player buys back in
	if p.Stack == 0 && amount < t.settings.MinBuy {
		sendError(c, fmt.Sprintf("rebuy must be at least %d", t.settings.MinBuy))
		return
	}
	if err := poker.BuyIn(t.game, pn, amount); err != nil {
		sendError(c, fmt.Sprintf("rebuy can't take your stack above %d", view.Config.MaxBuy))
		return
	}
	if !p.Ready && p.Stack == 0 {
		if err := poker.ToggleReady(t.game, pn, 0); err != nil {
			slog.Default().Warn("Toggle ready after rebuy", "error", err)
		}
	}
	t.record(p.Username, entryRebuy, amount, p.Username)
	t.send(createNewLog(fmt.Sprintf("%s rebuys for %d", p.Username, amount)))
	if view.Running && view.Stage == poker.PreDeal {
		t.scheduleDeal()
	}
	broadcastGame(t)
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
		applySitOuts(t)
		rebuyBots(t)
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
	delay := time.Duration(t.settings.NextHandDelay) * time.Millisecond
	t.dealTimer = time.AfterFunc(delay, func() { dealNextHand(t) })
}

// startTurnTimer gives the player to act the table's turn time to do so,
// unless it has none. Caller holds t.mu.
func (t *table) startTurnTimer() {
	t.turnSeq++
	seq := t.turnSeq
	if t.turnTimer != nil {
		t.turnTimer.Stop()
	}
	t.scheduleBot(seq)
	if t.settings.TurnTime == 0 {
		return
	}
	timeout := time.Duration(t.settings.TurnTime) * time.Millisecond
	t.turnDeadline = time.Now().Add(timeout)
	t.turnTimer = time.AfterFunc(timeout, func() { turnTimedOut(t, seq) })
}

// turnEndsIn is how many milliseconds the player to act has left, or 0 when
// nobody is betting or there's no turn timer. Caller holds t.mu.
func (t *table) turnEndsIn(view *poker.GameView) int64 {
	if !view.Betting || t.settings.TurnTime == 0 {
		return 0
	}
	return max(time.Until(t.turnDeadline).Milliseconds(), 0)
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
	if !isSeatedUUID(t, uuid) {
		return
	}
	view := t.game.GenerateOmniView()
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
			t.record(p.Username, entryCashOut, p.Stack, p.Username)
			t.send(createNewLog(fmt.Sprintf("%s left the table", p.Username)))
		}
		delete(t.leaving, uuid)
		delete(t.sitOutNext, uuid)
		delete(t.bots, uuid)
		for token, u := range t.tokens {
			if u == uuid {
				delete(t.tokens, token)
			}
		}
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
	sendErrorCode(c, "", message)
}

// sendErrorCode is sendError with a code the client can act on.
func sendErrorCode(c *Client, code, message string) {
	resp, err := json.Marshal(errorMessage{base{actionError}, code, message, currentTime()})
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
	return isSeatedUUID(c.table, c.id())
}

// isSeatedUUID reports whether a player with uuid is seated. Caller holds t.mu.
func isSeatedUUID(t *table, uuid string) bool {
	for _, p := range t.game.GenerateOmniView().Players {
		if p.UUID == uuid {
			return true
		}
	}
	return false
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
	view := t.game.GenerateOmniView()
	for i, p := range view.Players {
		u.byUUID[p.UUID] = createUpdatedGame(t, t.game.GeneratePlayerView(uint(i)), t.sitOutNext[p.UUID])
	}
	u.spectator = createUpdatedGame(t, t.game.GeneratePlayerView(uint(len(view.Players))), false)
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
	view := c.table.game.GeneratePlayerView(pn)
	return createUpdatedGame(c.table, view, ok && c.table.sitOutNext[c.id()])
}

// createUpdatedGame wraps one recipient's view of the game. Caller holds t.mu.
func createUpdatedGame(t *table, view *poker.GameView, sitOutNext bool) []byte {
	game := updateGame{
		base:        base{actionUpdateGame},
		Game:        view,
		Host:        t.hostID(),
		TurnEndsIn:  t.turnEndsIn(view),
		SitOutNext:  sitOutNext,
		Settings:    t.settings,
		HasPassword: t.password != "",
		Watchers:    t.watchers(),
		Bots:        t.botLevels(),
		Ledger:      t.ledgerView(),
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
		c.table.password,
	}
	resp, err := json.Marshal(joined)
	if err != nil {
		slog.Default().Warn("Marshal joined table", "error", err)
	}
	return resp
}

func createUpdatedPlayerUUID(c *Client, token string) []byte {
	uuid := updatePlayerUUID{
		base{actionUpdatePlayerUUID},
		c.id(),
		token,
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
