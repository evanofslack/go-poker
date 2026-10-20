package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestServer(t *testing.T) *httptest.Server {
	hub := newHub()
	go hub.run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
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
