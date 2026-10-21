package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024

	// close connections that send more than this many messages per second
	maxMessagesPerSecond = 20
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn // Websocket connection
	send     chan []byte     // Buffered channel of outbound bytes
	uuid     atomic.Value    // string, read by the table goroutine; use id()
	username string
	table    *table // Player's table
}

func newClient(conn *websocket.Conn, hub *Hub) *Client {
	c := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 1024),
	}
	c.uuid.Store(newUUID())
	return c
}

func newUUID() string {
	return uuid.New().String()
}

// id is the client's UUID, which becomes its player's UUID once seated.
func (c *Client) id() string {
	return c.uuid.Load().(string)
}

func (c *Client) disconnect() {
	// leave the table before the hub closes c.send, so the table never
	// sends on a closed channel
	c.hub.leaveTable(c, true)
	c.hub.unregister <- c
	c.conn.Close()
}

// readPump pumps events from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.disconnect()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Default().Warn("set read deadline", "error", err)
	}
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	windowStart, count := time.Now(), 0
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Default().Warn("Websocket unexpected close", "error", err)
			}
			slog.Default().Warn("Read from websocket", "error", err)
			break
		}
		if time.Since(windowStart) > time.Second {
			windowStart, count = time.Now(), 0
		}
		count++
		if count > maxMessagesPerSecond {
			slog.Default().Warn("Websocket rate limit exceeded, closing", "uuid", c.id())
			break
		}
		if err = c.safeProcessEvents(message); err != nil {
			slog.Default().Warn("Process websocket message", "error", err)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				slog.Default().Warn("Write websocket message", "error", err)
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Default().Warn("Write websocket ping", "error", err)
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := newClient(conn, hub)

	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// safeProcessEvents recovers from a panic while handling a message, so one bad
// message cannot take down the server.
func (c *Client) safeProcessEvents(rawMessage []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic processing message: %v", r)
		}
	}()
	return c.processEvents(rawMessage)
}

// closedWhenEnded are the actions that would deal, seat or move chips, which
// stop once a session ends so its ledger stays final.
var closedWhenEnded = map[string]bool{
	actionTakeSeat:     true,
	actionStartGame:    true,
	actionRebuy:        true,
	actionSitOut:       true,
	actionSitOutPlayer: true,
	actionApproveBuy:   true,
	actionDenyBuy:      true,
	actionAdjustChips:  true,
	actionPauseGame:    true,
	actionResumeGame:   true,
	actionEndSession:   true,
	actionAddBot:       true,
	actionRemoveBot:    true,
}

func (c *Client) processEvents(rawMessage []byte) error {
	var baseMessage base
	err := json.Unmarshal(rawMessage, &baseMessage)
	if err != nil {
		return err
	}

	if baseMessage.Action == "" {
		return errors.New("deserialize message")
	}

	joining := baseMessage.Action == actionCreateTable || baseMessage.Action == actionJoinTable
	if !joining && baseMessage.Action != actionListTables && c.table == nil {
		return errors.New("client has not joined a table")
	}
	if closedWhenEnded[baseMessage.Action] && c.table.ended() {
		sendError(c, "the session has ended")
		return nil
	}

	switch baseMessage.Action {

	case actionCreateTable, actionJoinTable:
		var table joinTable
		err := json.Unmarshal(rawMessage, &table)
		if err != nil {
			return err
		}
		username := strings.TrimSpace(table.Username)
		if !validLength(table.Tablename, maxTablenameLen) {
			sendError(c, "invalid table name")
			return errors.New("invalid table name")
		}
		if !validUsername(username) {
			sendError(c, "invalid name")
			return errors.New("invalid username")
		}
		create := baseMessage.Action == actionCreateTable
		if create && !validPassword(table.Password) {
			sendError(c, "invalid password")
			return errors.New("invalid password")
		}
		settings := defaultSettings
		if create && table.Settings != nil {
			if err := json.Unmarshal(table.Settings, &settings); err != nil {
				sendError(c, "invalid settings")
				return err
			}
			if err := validSettings(settings); err != nil {
				sendError(c, err.Error())
				return err
			}
		}
		handleJoinTable(c, table.Tablename, username, table.Token, table.Password, create, settings)
		return nil

	case actionListTables:
		// carries nothing to validate; readPump's rate limit covers polling
		c.send <- createTables(c.hub.publicTables())
		return nil

	case actionLeaveTable:
		var table leaveTable
		err := json.Unmarshal(rawMessage, &table)
		if err != nil {
			return err
		}
		handleLeaveTable(c, table.Tablename)
		return nil

	case actionSendMessage:
		var message sendMessage
		err := json.Unmarshal(rawMessage, &message)
		if err != nil {
			return err
		}
		if !validLength(message.Message, maxMessageLen) {
			return errors.New("invalid chat message")
		}
		handleSendMessage(c, message.Message)
		return nil

	case actionTakeSeat:
		var seat takeSeat
		err := json.Unmarshal(rawMessage, &seat)
		if err != nil {
			return err
		}
		handleTakeSeat(c, seat.SeatID, seat.BuyIn)
		return nil

	case actionStartGame:
		handleStartGame(c)
		return nil

	case actionPlayerCall:
		handleCall(c)
		return nil

	case actionPlayerCheck:
		handleCheck(c)
		return nil

	case actionPlayerRaise:
		var raise playerRaise
		err := json.Unmarshal(rawMessage, &raise)
		if err != nil {
			return err
		}
		handleRaise(c, raise.Amount)
		return nil

	case actionPlayerFold:
		handleFold(c)
		return nil

	case actionLeaveSeat:
		handleLeaveSeat(c)
		return nil

	case actionSitOut:
		handleSitOut(c)
		return nil

	case actionRebuy:
		var r rebuy
		err := json.Unmarshal(rawMessage, &r)
		if err != nil {
			return err
		}
		if !validAmount(r.Amount) {
			sendError(c, "rebuy amount must be more than 0")
			return errors.New("invalid rebuy amount")
		}
		handleRebuy(c, r.Amount)
		return nil

	case actionAddBot:
		var b addBot
		err := json.Unmarshal(rawMessage, &b)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(b.Name)
		if !validUsername(name) {
			sendError(c, "invalid bot name")
			return errors.New("invalid bot name")
		}
		if !validBotLevel(b.Level) {
			sendError(c, "invalid bot level")
			return errors.New("invalid bot level")
		}
		handleAddBot(c, b.SeatID, b.BuyIn, name, b.Level)
		return nil

	case actionRemoveBot:
		var b removeBot
		err := json.Unmarshal(rawMessage, &b)
		if err != nil {
			return err
		}
		handleRemoveBot(c, b.SeatID)
		return nil

	case actionMakeHost, actionSetBanker, actionMoveToSpectator, actionRemovePlayer, actionSitOutPlayer,
		actionApproveBuy, actionDenyBuy:
		var target playerTarget
		err := json.Unmarshal(rawMessage, &target)
		if err != nil {
			return err
		}
		if !validLength(target.Name, maxUsernameLen) {
			sendError(c, "invalid name")
			return errors.New("invalid host action name")
		}
		hostActions[baseMessage.Action](c, target.Name)
		return nil

	case actionPauseGame, actionResumeGame:
		handlePause(c, baseMessage.Action == actionPauseGame)
		return nil

	case actionEndSession:
		handleEndSession(c)
		return nil

	case actionAdjustChips:
		var adjust adjustChips
		err := json.Unmarshal(rawMessage, &adjust)
		if err != nil {
			return err
		}
		if !validLength(adjust.Name, maxUsernameLen) || !validAdjust(adjust.Amount) || !validReason(adjust.Reason) {
			sendError(c, "invalid chip change")
			return errors.New("invalid chip change")
		}
		handleAdjustChips(c, adjust.Name, adjust.Amount, adjust.Reason)
		return nil

	default:
		return errors.New("unexpected message action")
	}
}
