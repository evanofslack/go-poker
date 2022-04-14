package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn // Websocket connection
	send     chan []byte     // Buffered channel of outbound bytes
	uuid     string          // UUID
	username string
	seatID   uint   // Seat number
	table    *table // Player's table
}

func newClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 1024),
		uuid: uuid.New().String(),
	}
}

func (client *Client) disconnect() {
	client.hub.unregister <- client
	client.table.unregister <- client
}

// readPump pumps events from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		err = c.processEvents(message)
		if err != nil {
			fmt.Println(err)
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
				log.Printf("error: %v", err)
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
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

func (c *Client) processEvents(rawMessage []byte) error {
	var baseMessage base
	err := json.Unmarshal(rawMessage, &baseMessage)
	if err != nil {
		return err
	}

	if baseMessage.Action == "" {
		return errors.New("error deserializing message")
	}

	switch baseMessage.Action {

	case actionJoinTable:
		var table joinTable
		err := json.Unmarshal(rawMessage, &table)
		if err != nil {
			return err
		}
		handleJoinTable(c, table.Tablename)
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
		handleSendMessage(c, message.Username, message.Message)
		return nil

	case actionNewPlayer:
		var player newPlayer
		err := json.Unmarshal(rawMessage, &player)
		if err != nil {
			return err
		}
		handleNewPlayer(c, player.Username)
		return nil

	case actionTakeSeat:
		var seat takeSeat
		err := json.Unmarshal(rawMessage, &seat)
		if err != nil {
			return err
		}
		handleTakeSeat(c, seat.Username, seat.SeatID, seat.BuyIn)
		return nil

	case actionStartGame:
		handleStartGame(c)
		return nil

	case actionResetGame:
		handleResetGame(c)
		return nil

	case actionDealGame:
		handleDealGame(c)
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

	default:
		return errors.New("unexpected message action")
	}

}
