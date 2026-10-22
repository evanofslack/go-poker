package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/holdem"
	"github.com/evanofslack/go-poker/table"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

// close connections that send more than this many messages per second
var maxMessagesPerSecond = 20

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub  *Hub
	conn *websocket.Conn // Websocket connection
	send chan []byte     // Buffered channel of outbound bytes
	quit chan struct{}   // closed by the hub on shutdown
	done chan struct{}   // closed when writePump returns
	ip   string          // remote address, counted against the per-ip limits

	// owned by readPump
	table  *tableActor // the table joined, nil before
	person string      // who the client is at table

}

func newClient(conn *websocket.Conn, hub *Hub, ip string) *Client {
	c := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 1024),
		ip:   ip,
		quit: make(chan struct{}),
		done: make(chan struct{}),
	}
	return c
}

func (c *Client) disconnect() {
	// leave the table before the hub closes c.send, so the table never
	// sends on a closed channel
	if c.table != nil {
		c.table.ask(table.Disconnect{Person: c.person}, c)
	}
	c.hub.unregister <- c
	c.conn.Close()
	c.hub.releaseConn(c.ip)
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
			slog.Default().Warn("Websocket rate limit exceeded, closing", "ip", c.ip)
			limitHits.WithLabelValues("messages").Inc()
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
		close(c.done)
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
			if err := c.write(message); err != nil {
				return
			}

		case <-c.quit:
			// the server is shutting down: flush what's queued, then close
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			for len(c.send) > 0 {
				if err := c.write(<-c.send); err != nil {
					return
				}
			}
			c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "server restarting"))
			return

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Default().Warn("Write websocket ping", "error", err)
				return
			}
		}
	}
}

func (c *Client) write(message []byte) error {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		slog.Default().Warn("Write websocket message", "error", err)
		return err
	}
	w.Write(message)
	return w.Close()
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r, trustProxy)
	if ok, limit := hub.reserveConn(ip); !ok {
		limitHits.WithLabelValues(limit).Inc()
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.releaseConn(ip)
		log.Println(err)
		return
	}
	client := newClient(conn, hub, ip)

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
			panicsTotal.Inc()
			err = fmt.Errorf("panic processing message: %v", r)
		}
	}()
	return c.processEvents(rawMessage)
}

// sendError tells c why its message was refused.
func (c *Client) sendError(code, msg string) {
	select {
	case c.send <- marshal(ErrorOut{ActionError, code, msg}):
	default:
	}
}

// refuse tells c why its message was refused, and returns that as an error.
func (c *Client) refuse(msg string) error {
	c.sendError("", msg)
	return errors.New(msg)
}

// hostKinds are the host actions that name a player.
var hostKinds = map[string]table.HostKind{
	ActionMakeHost:        table.MakeHost,
	ActionSetBanker:       table.SetBanker,
	ActionMoveToSpectator: table.MoveToSpectator,
	ActionRemovePlayer:    table.RemovePlayer,
	ActionSitOutPlayer:    table.SitOutPlayer,
	ActionApproveBuy:      table.ApproveBuy,
	ActionDenyBuy:         table.DenyBuy,
}

func (c *Client) processEvents(raw []byte) error {
	var in Inbound
	if err := json.Unmarshal(raw, &in); err != nil {
		return err
	}
	switch in.Action {
	case "":
		return errors.New("deserialize message")
	case ActionCreateTable, ActionJoinTable:
		return c.join(in, in.Action == ActionCreateTable)
	case ActionListTables:
		// carries nothing to validate; readPump's rate limit covers polling
		c.send <- marshal(Tables{ActionTables, c.hub.publicTables()})
		return nil
	}

	if c.table == nil {
		return errors.New("client has not joined a table")
	}
	cmd, err := c.command(in)
	if err != nil {
		return err
	}
	r := c.table.ask(cmd, c)
	if _, ok := cmd.(table.Leave); ok || errors.Is(r.err, errClosed) {
		c.table, c.person = nil, ""
	}
	return r.err
}

// command is the table command for in, from c's person.
func (c *Client) command(in Inbound) (table.Command, error) {
	p := c.person
	switch in.Action {
	case ActionLeaveTable:
		return table.Leave{Person: p}, nil
	case ActionSendMessage:
		if !validLength(in.Message, maxMessageLen) {
			return nil, errors.New("invalid chat message")
		}
		return table.Chat{Person: p, Text: in.Message}, nil
	case ActionTakeSeat:
		if !validBuyIn(in.Amount) {
			return nil, c.refuse("invalid buy-in")
		}
		return table.TakeSeat{Person: p, Seat: in.Seat, BuyIn: uint(in.Amount)}, nil
	case ActionStartGame:
		return table.Start{Person: p}, nil
	case ActionCheck:
		return table.Play{Person: p, Action: holdem.Check{}}, nil
	case ActionCall:
		return table.Play{Person: p, Action: holdem.Call{}}, nil
	case ActionFold:
		return table.Play{Person: p, Action: holdem.Fold{}}, nil
	case ActionRaise:
		return table.Play{Person: p, Action: holdem.Raise{To: in.To}}, nil
	case ActionGetHands:
		if !validHandNumber(in.Before) {
			return nil, errors.New("invalid hand number")
		}
		return table.GetHands{Person: p, Before: in.Before}, nil
	case ActionLeaveSeat:
		return table.LeaveSeat{Person: p}, nil
	case ActionSitOut:
		return table.SitOut{Person: p}, nil
	case ActionRebuy:
		if !validBuyIn(in.Amount) {
			return nil, c.refuse("rebuy amount must be more than 0")
		}
		return table.Rebuy{Person: p, Amount: uint(in.Amount)}, nil
	case ActionAddBot:
		name := strings.TrimSpace(in.Name)
		switch {
		case !validUsername(name):
			return nil, c.refuse("invalid bot name")
		case !validBotLevel(in.Level):
			return nil, c.refuse("invalid bot level")
		case !validBuyIn(in.Amount):
			return nil, c.refuse("invalid buy-in")
		}
		return table.AddBot{Person: p, Seat: in.Seat, BuyIn: uint(in.Amount), Name: name, Level: in.Level}, nil
	case ActionRemoveBot:
		return table.RemoveBot{Person: p, Seat: in.Seat}, nil
	case ActionMakeHost, ActionSetBanker, ActionMoveToSpectator, ActionRemovePlayer, ActionSitOutPlayer,
		ActionApproveBuy, ActionDenyBuy:
		if !validLength(in.Name, maxUsernameLen) {
			return nil, c.refuse("invalid name")
		}
		return table.HostAction{Person: p, Kind: hostKinds[in.Action], Name: in.Name}, nil
	case ActionPauseGame, ActionResumeGame:
		return table.Pause{Person: p, Pause: in.Action == ActionPauseGame}, nil
	case ActionEndSession:
		return table.EndSession{Person: p}, nil
	case ActionAdjustChips:
		if !validLength(in.Name, maxUsernameLen) || !validAdjust(in.Amount) || !validReason(in.Reason) {
			return nil, c.refuse("invalid chip change")
		}
		return table.AdjustChips{Person: p, Name: in.Name, Amount: in.Amount, Reason: in.Reason}, nil
	}
	return nil, errors.New("unexpected message action")
}

// join creates or joins the table in names, leaving any table c is at.
func (c *Client) join(in Inbound, create bool) error {
	username := strings.TrimSpace(in.Username)
	if !validLength(in.Tablename, maxTablenameLen) {
		return c.refuse("invalid table name")
	}
	if !validUsername(username) {
		return c.refuse("invalid name")
	}
	if create && !validPassword(in.Password) {
		return c.refuse("invalid password")
	}
	settings := table.DefaultSettings
	if create && in.Settings != nil {
		if err := json.Unmarshal(in.Settings, &settings); err != nil {
			return c.refuse("invalid settings")
		}
		if err := settings.Validate(); err != nil {
			return c.refuse(err.Error())
		}
	}

	if c.table != nil {
		c.table.ask(table.Leave{Person: c.person}, c)
		c.table, c.person = nil, ""
	}
	var a *tableActor
	var err error
	if create {
		a, err = c.hub.createTable(in.Tablename, c.ip, in.Password, settings)
	} else if a = c.hub.table(in.Tablename); a == nil {
		err = errNoTable
	}
	if err != nil {
		return c.refuse(err.Error())
	}
	r := a.ask(table.Join{ID: uuid.New().String(), Name: username, Token: in.Token, Password: in.Password}, c)
	if r.err != nil {
		return c.refuse(r.err.Error())
	}
	if r.joined != nil {
		c.table, c.person = a, r.joined.ID
	}
	return nil
}

// clientIP is the address the request came from. Behind a trusted proxy it is
// the last X-Forwarded-For entry, the one the proxy added; earlier entries
// come from the client and can be made up.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
