package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// readUntil reads conn until it gets action, returning its code and message.
// With action "", it reads until the server closes conn.
func readUntil(t *testing.T, conn *websocket.Conn, action string) (string, string) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var code, message string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if action == "" && websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return code, message
			}
			t.Fatalf("waiting for %q: %s", action, err)
		}
		var m struct{ Action, Code, Message string }
		json.Unmarshal(msg, &m)
		if m.Action == actionError {
			code, message = m.Code, m.Message
		}
		if action != "" && m.Action == action {
			return code, message
		}
	}
}

func getStatus(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestShutdownClosesTables(t *testing.T) {
	s := &Server{router: chi.NewRouter(), hub: newHub(), server: &http.Server{}}
	s.ready.Store(true)
	s.mountStatus()
	go s.hub.run()
	mux := http.NewServeMux()
	mux.Handle("/readyz", s.router)
	mux.HandleFunc("/", s.serveWebsocket)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if code := getStatus(t, srv.URL+"/readyz"); code != http.StatusOK {
		t.Fatalf("readyz = %d before shutdown", code)
	}
	a := dialRaw(t, srv)
	send(t, a, createMsg("t", "a"))
	readUntil(t, a, actionUpdateGame)
	b := dialRaw(t, srv)
	send(t, b, joinMsg("t", "b"))
	readUntil(t, b, actionUpdateGame)
	lobby := dialRaw(t, srv)
	send(t, lobby, `{"action":"list-tables"}`)
	readUntil(t, lobby, actionTables)

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*websocket.Conn{a, b} {
		code, msg := readUntil(t, conn, "")
		if code != codeClosed || msg != "the server is restarting" {
			t.Errorf("got error %q %q, want %q %q", code, msg, codeClosed, "the server is restarting")
		}
	}
	readUntil(t, lobby, "")
	waitForTables(t, s.hub)
	if code := getStatus(t, srv.URL+"/readyz"); code != http.StatusServiceUnavailable {
		t.Errorf("readyz = %d after shutdown", code)
	}
}
