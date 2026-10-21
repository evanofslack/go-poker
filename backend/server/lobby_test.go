package server

import (
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// createWithMsg is createMsg with settings and a password
func createWithMsg(table, name, settings, password string) string {
	return `{"action":"create-table","tablename":"` + table + `","username":"` + name +
		`","settings":` + settings + `,"password":"` + password + `"}`
}

// publicSettings lists a table, with timers that pass validation
const publicSettings = `{"public":true,"turnTime":0,"nextHandDelay":2000}`

// listTables asks for the public tables as a client that hasn't joined one,
// returning the reply as sent.
func listTables(t *testing.T, conn *websocket.Conn) (tablesMessage, string) {
	t.Helper()
	send(t, conn, `{"action":"list-tables"}`)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var msg tablesMessage
		if json.Unmarshal(raw, &msg) == nil && msg.Action == actionTables {
			return msg, string(raw)
		}
	}
}

func summaryNames(tables []tableSummary) []string {
	names := []string{}
	for _, s := range tables {
		names = append(names, s.Name)
	}
	return names
}

// dialRaw is dial without draining, for reading replies directly
func dialRaw(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestListTablesBeforeJoining(t *testing.T) {
	srv := newTestServer(t)
	msg, _ := listTables(t, dialRaw(t, srv))
	if msg.Tables == nil || len(msg.Tables) != 0 {
		t.Fatalf("tables = %v, want an empty list", msg.Tables)
	}
}

func TestPublicTables(t *testing.T) {
	srv, hub := newTestServerHub(t)

	// full: a player and a bot in two seats
	full := dial(t, srv)
	send(t, full, createWithMsg("full", "ann", `{"public":true,"seats":2,"turnTime":0,"nextHandDelay":2000}`, ""), seatMsg(1), addBotMsg(2, "bob"))
	// open: a player and a watcher
	open := dial(t, srv)
	send(t, open, createWithMsg("open", "cat", publicSettings, ""), seatMsg(1))
	// locked: nobody seated, so its host is watching
	locked := dial(t, srv)
	send(t, locked, createWithMsg("locked", "dan", publicSettings, "hunter2"))
	// private and ended tables aren't listed
	private := dial(t, srv)
	send(t, private, createMsg("private", "eve"))
	ended := dial(t, srv)
	send(t, ended, createWithMsg("ended", "fay", publicSettings, ""))
	waitForTables(t, hub, "ended", "full", "locked", "open", "private")
	watcher := dial(t, srv)
	send(t, watcher, joinMsg("open", "gus"))

	hub.mu.Lock()
	done := hub.tables["ended"]
	hub.mu.Unlock()
	done.mu.Lock()
	done.endedAt = time.Now()
	done.mu.Unlock()

	// the bot and watcher arrive asynchronously
	var tables []tableSummary
	eventually(t, func() bool {
		tables = hub.publicTables()
		return len(tables) == 3 && tables[0].Watchers == 1 && tables[2].Bots == 1
	}, "tables = %+v", tables)

	if got, want := summaryNames(tables), []string{"open", "locked", "full"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	want := []tableSummary{
		{Name: "open", Players: 1, Watchers: 1},
		{Name: "locked", Watchers: 1, HasPassword: true},
		{Name: "full", Players: 2, Bots: 1, Seats: 2},
	}
	for i, w := range want {
		g := tables[i]
		if g.Players != w.Players || g.Bots != w.Bots || g.Watchers != w.Watchers || g.HasPassword != w.HasPassword {
			t.Errorf("%s = %+v, want %+v", g.Name, g, w)
		}
		if w.Seats != 0 && g.Seats != w.Seats {
			t.Errorf("%s seats = %d, want %d", g.Name, g.Seats, w.Seats)
		}
	}
	if g := tables[0]; g.BigBlind != defaultSettings.BigBlind || g.MinBuy != defaultSettings.MinBuy {
		t.Errorf("open settings = %+v, want the defaults", g)
	}

	// the reply over the socket never carries the password
	_, raw := listTables(t, dialRaw(t, srv))
	if strings.Contains(raw, "hunter2") {
		t.Fatalf("reply leaks the password: %s", raw)
	}
}
