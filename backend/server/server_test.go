package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/gorilla/websocket"
)

func newTestServer(t *testing.T) *httptest.Server {
	srv, _ := newTestServerHub(t)
	return srv
}

func newTestServerHub(t *testing.T) (*httptest.Server, *Hub) {
	hub := newHub()
	go hub.run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, hub
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	// drain responses so the server never blocks on us
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return conn
}

func send(t *testing.T, conn *websocket.Conn, msgs ...string) {
	for _, m := range msgs {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
			t.Error(err) // not Fatal: also called from other goroutines
			return
		}
	}
}

// assertAlive checks a fresh client can still join and get a game update.
func assertAlive(t *testing.T, srv *httptest.Server) {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("server not accepting connections: %s", err)
	}
	defer conn.Close()
	send(t, conn, `{"action":"join-table","tablename":"alive"}`, `{"action":"new-player","username":"probe"}`)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("no game update from server: %s", err)
		}
		if strings.Contains(string(msg), `"update-game"`) {
			return
		}
	}
}

func TestMalformedMessagesDoNotCrash(t *testing.T) {
	srv := newTestServer(t)
	conn := dial(t, srv)

	send(t, conn,
		// actions before joining a table
		`{"action":"send-message","username":"a","message":"hi"}`,
		`{"action":"take-seat","username":"a","seatID":1,"buyIn":100}`,
		`{"action":"start-game"}`,
		`{"action":"leave-table","tablename":"nope"}`,
		`not json`,
		// actions at an empty table
		`{"action":"join-table","tablename":"t"}`,
		`{"action":"start-game"}`,
		`{"action":"deal-game"}`,
		`{"action":"player-fold"}`,
		`{"action":"player-call"}`,
		`{"action":"reset-game"}`,
		// bad seats
		`{"action":"take-seat","username":"a","seatID":0,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":99,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":1,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":1,"buyIn":100}`,
		// switch tables without leaving
		`{"action":"join-table","tablename":"other"}`,
	)
	time.Sleep(200 * time.Millisecond)
	assertAlive(t, srv)
}

func TestConcurrentPlayers(t *testing.T) {
	srv := newTestServer(t)

	var wg sync.WaitGroup
	for i := 1; i <= 4; i++ {
		conn := dial(t, srv)
		seat := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			send(t, conn,
				`{"action":"join-table","tablename":"t"}`,
				`{"action":"new-player","username":"p"}`,
				`{"action":"take-seat","username":"p","seatID":`+string(rune('0'+seat))+`,"buyIn":100}`,
			)
			for j := 0; j < 2; j++ { // stay under the rate limit
				send(t, conn,
					`{"action":"start-game"}`,
					`{"action":"player-call"}`,
					`{"action":"player-check"}`,
					`{"action":"player-raise","amount":40}`,
					`{"action":"player-fold"}`,
					`{"action":"deal-game"}`,
					`{"action":"send-message","username":"p","message":"hi"}`,
				)
			}
		}()
	}
	wg.Wait()
	time.Sleep(200 * time.Millisecond)
	assertAlive(t, srv)
}

func TestDisconnectMidGame(t *testing.T) {
	srv := newTestServer(t)
	a := dial(t, srv)
	b := dial(t, srv)
	for i, c := range []*websocket.Conn{a, b} {
		send(t, c,
			`{"action":"join-table","tablename":"t"}`,
			`{"action":"new-player","username":"p"}`,
			`{"action":"take-seat","username":"p","seatID":`+string(rune('1'+i))+`,"buyIn":100}`,
		)
	}
	send(t, a, `{"action":"start-game"}`)
	a.Close()
	for i := 0; i < 10; i++ {
		send(t, b, `{"action":"send-message","username":"p","message":"still here"}`)
	}
	time.Sleep(200 * time.Millisecond)
	assertAlive(t, srv)
}

func TestCrossOriginRejected(t *testing.T) {
	srv := newTestServer(t)
	url := "ws" + strings.TrimPrefix(srv.URL, "http")

	_, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"https://evil.example"}})
	if err == nil {
		t.Fatal("cross-origin websocket was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Errorf("want 403, got %v", resp)
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {srv.URL}})
	if err != nil {
		t.Fatalf("same-origin websocket rejected: %s", err)
	}
	conn.Close()
}

func TestFloodDisconnects(t *testing.T) {
	srv := newTestServer(t)
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	send(t, conn, `{"action":"join-table","tablename":"t"}`)
	for i := 0; i < 2*maxMessagesPerSecond; i++ {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"send-message","username":"p","message":"spam"}`))
	}
	// the server should close the connection, so writes start failing
	deadline := time.Now().Add(2 * time.Second)
	for conn.WriteMessage(websocket.TextMessage, []byte(`{}`)) == nil {
		if time.Now().After(deadline) {
			t.Fatal("flooding client was not disconnected")
		}
		time.Sleep(10 * time.Millisecond)
	}
	assertAlive(t, srv)
}

func tableNames(h *Hub) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var names []string
	for name := range h.tables {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// eventually polls cond until it is true, failing the test after 2 seconds.
func eventually(t *testing.T, cond func() bool, msg string, args ...any) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf(msg, args...)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForTables waits until exactly the wanted tables exist.
func waitForTables(t *testing.T, h *Hub, want ...string) {
	t.Helper()
	eventually(t, func() bool { return slices.Equal(tableNames(h), want) }, "tables = %v, want %v", tableNames(h), want)
}

func TestEmptyTablesDeleted(t *testing.T) {
	srv, hub := newTestServerHub(t)
	conn := dial(t, srv)

	send(t, conn, `{"action":"join-table","tablename":"a"}`, `{"action":"join-table","tablename":"b"}`)
	waitForTables(t, hub, "b")

	send(t, conn, `{"action":"leave-table","tablename":"b"}`)
	waitForTables(t, hub)

	send(t, conn, `{"action":"join-table","tablename":"c"}`)
	waitForTables(t, hub, "c")
	conn.Close()
	waitForTables(t, hub)
}

func TestTableCap(t *testing.T) {
	srv, hub := newTestServerHub(t)
	for i := 0; i < maxTables+5; i++ {
		conn := dial(t, srv)
		send(t, conn, `{"action":"join-table","tablename":"t`+strconv.Itoa(i)+`"}`)
	}
	eventually(t, func() bool { return len(tableNames(hub)) >= maxTables }, "tables never reached the cap")
	time.Sleep(100 * time.Millisecond) // give extra joins time to (wrongly) succeed
	if n := len(tableNames(hub)); n != maxTables {
		t.Errorf("tables = %d, want %d", n, maxTables)
	}
}

func isRunning(t *testing.T, h *Hub, name string) bool {
	h.mu.Lock()
	tbl := h.tables[name]
	h.mu.Unlock()
	if tbl == nil {
		t.Fatalf("no table %q", name)
	}
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	return tbl.game.GenerateOmniView().Running
}

func TestSpectatorCannotStart(t *testing.T) {
	srv, hub := newTestServerHub(t)
	for i := 1; i <= 2; i++ {
		send(t, dial(t, srv),
			`{"action":"join-table","tablename":"t"}`,
			`{"action":"new-player","username":"p"}`,
			`{"action":"take-seat","username":"p","seatID":`+strconv.Itoa(i)+`,"buyIn":100}`,
		)
	}
	spectator := dial(t, srv)
	send(t, spectator, `{"action":"join-table","tablename":"t"}`, `{"action":"start-game"}`)
	time.Sleep(200 * time.Millisecond)
	if isRunning(t, hub, "t") {
		t.Fatal("spectator started the game")
	}

	player := dial(t, srv)
	send(t, player,
		`{"action":"join-table","tablename":"t"}`,
		`{"action":"new-player","username":"p"}`,
		`{"action":"take-seat","username":"p","seatID":3,"buyIn":100}`,
		`{"action":"start-game"}`,
	)
	eventually(t, func() bool { return isRunning(t, hub, "t") }, "seated player could not start the game")
}

// received collects what the server sends one client.
type received struct {
	games  chan poker.GameView
	errors chan string
}

// dialRecv is dial, but also collects the game views and errors the client receives.
func dialRecv(t *testing.T, srv *httptest.Server) (*websocket.Conn, received) {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	r := received{make(chan poker.GameView, 256), make(chan string, 256)}
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var u struct {
				Action  string         `json:"action"`
				Game    poker.GameView `json:"game"`
				Message string         `json:"message"`
			}
			if json.Unmarshal(msg, &u) != nil {
				continue
			}
			switch u.Action {
			case actionUpdateGame:
				select {
				case r.games <- u.Game:
				default:
				}
			case actionError:
				select {
				case r.errors <- u.Message:
				default:
				}
			}
		}
	}()
	return conn, r
}

// waitGame returns the first game view matching cond.
func waitGame(t *testing.T, games <-chan poker.GameView, cond func(poker.GameView) bool) poker.GameView {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case g := <-games:
			if cond(g) {
				return g
			}
		case <-timeout:
			t.Fatal("timed out waiting for game update")
		}
	}
}

func TestHoleCardsHidden(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	b, bRecv := dialRecv(t, srv)
	spectator, sRecv := dialRecv(t, srv)
	for i, c := range []*websocket.Conn{a, b} {
		send(t, c,
			`{"action":"join-table","tablename":"t"}`,
			`{"action":"new-player","username":"p"}`,
			`{"action":"take-seat","username":"p","seatID":`+strconv.Itoa(i+1)+`,"buyIn":100}`,
		)
	}
	send(t, spectator, `{"action":"join-table","tablename":"t"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, a, `{"action":"start-game"}`)

	running := func(g poker.GameView) bool { return g.Running && g.Players[0].In }
	views := map[string]poker.GameView{
		"a":         waitGame(t, aRecv.games, running),
		"b":         waitGame(t, bRecv.games, running),
		"spectator": waitGame(t, sRecv.games, running),
	}
	own := map[string]int{"a": 0, "b": 1, "spectator": -1}
	for name, g := range views {
		for pn, p := range g.Players {
			if shown, want := p.Cards[0] != 0, pn == own[name]; shown != want {
				t.Errorf("%s sees player %d cards = %v, want %v", name, pn, shown, want)
			}
		}
	}
}

// waitError returns the next error the client receives.
func waitError(t *testing.T, r received) string {
	t.Helper()
	select {
	case msg := <-r.errors:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error")
		return ""
	}
}

func TestOnlyActOnYourTurn(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	b, bRecv := dialRecv(t, srv)
	spectator, sRecv := dialRecv(t, srv)
	for i, c := range []*websocket.Conn{a, b} {
		send(t, c,
			`{"action":"join-table","tablename":"t"}`,
			`{"action":"new-player","username":"p"}`,
			`{"action":"take-seat","username":"p","seatID":`+strconv.Itoa(i+1)+`,"buyIn":100}`,
		)
	}
	send(t, spectator, `{"action":"join-table","tablename":"t"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, a, `{"action":"start-game"}`)
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting })

	// heads up, the dealer (seat 1, a) acts first preflop
	if g.ActionNum != 0 {
		t.Fatalf("action on player %d, want 0", g.ActionNum)
	}
	send(t, b, `{"action":"player-fold"}`)
	if msg := waitError(t, bRecv); msg != "not your turn" {
		t.Errorf("b got error %q", msg)
	}
	send(t, spectator, `{"action":"player-fold"}`)
	if msg := waitError(t, sRecv); msg != "take a seat first" {
		t.Errorf("spectator got error %q", msg)
	}

	// neither fold happened: a can still act, and the hand goes on
	send(t, a, `{"action":"player-call"}`)
	g = waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.ActionNum == 1 })
	if !g.Players[1].In {
		t.Error("b was folded by someone else")
	}
}

func TestSeatErrors(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	b, bRecv := dialRecv(t, srv)
	for _, c := range []*websocket.Conn{a, b} {
		send(t, c, `{"action":"join-table","tablename":"t"}`, `{"action":"new-player","username":"p"}`)
	}
	send(t, a, `{"action":"take-seat","username":"p","seatID":1,"buyIn":100}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })

	send(t, a, `{"action":"take-seat","username":"p","seatID":2,"buyIn":100}`)
	if msg := waitError(t, aRecv); msg != "you already have a seat" {
		t.Errorf("second seat: got %q", msg)
	}
	send(t, b, `{"action":"take-seat","username":"p","seatID":1,"buyIn":100}`)
	if msg := waitError(t, bRecv); msg != "seat taken" {
		t.Errorf("taken seat: got %q", msg)
	}
	send(t, b, `{"action":"take-seat","username":"p","seatID":2,"buyIn":999999}`)
	if msg := waitError(t, bRecv); !strings.HasPrefix(msg, "buy-in must be") {
		t.Errorf("bad buy-in: got %q", msg)
	}
}
