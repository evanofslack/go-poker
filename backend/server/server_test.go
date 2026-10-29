package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/table"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMain(m *testing.M) {
	timing = table.Timing{
		ReconnectGrace: 300 * time.Millisecond,
		EndedTTL:       500 * time.Millisecond,
		IdleTimeout:    time.Hour,
		StreetDelay:    10 * time.Millisecond,
		BotDelay:       5 * time.Millisecond,
	}
	// tests connect and create tables from one address
	maxConnsPerIP = 1000
	maxTablesPerIP = 1000
	os.Exit(m.Run())
}

func newTestServer(t *testing.T) *httptest.Server {
	srv, _ := newTestServerHub(t, newMemStore())
	return srv
}

func newTestServerHub(t *testing.T, store Store) (*httptest.Server, *Hub) {
	hub := newHub(store)
	go hub.run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, hub
}

// msg is a message from the server.
type msg struct {
	Action string `json:"action"`
	raw    []byte
}

func (m msg) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(m.raw, v); err != nil {
		t.Fatal(err)
	}
}

// client is a test connection that keeps what the server sends.
type client struct {
	t       *testing.T
	conn    *websocket.Conn
	msgs    chan msg
	timeout time.Duration // how long wait waits
}

func dial(t *testing.T, srv *httptest.Server) *client {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	c := &client{t, conn, make(chan msg, 1000), 2 * time.Second}
	go func() {
		defer close(c.msgs)
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				return
			}
			m := msg{raw: b}
			json.Unmarshal(b, &m)
			select {
			case c.msgs <- m:
			default: // nobody is reading; drop it rather than stall
			}
		}
	}()
	return c
}

func (c *client) send(msgs ...string) {
	for _, m := range msgs {
		if err := c.conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
			c.t.Error(err) // not Fatal: also called from other goroutines
			return
		}
	}
}

// wait reads messages until one with action passes cond, failing after
// c.timeout.
func (c *client) wait(action string, cond func(msg) bool) msg {
	c.t.Helper()
	timeout := time.After(c.timeout)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				c.t.Fatalf("connection closed waiting for %s", action)
			}
			if m.Action == action && (cond == nil || cond(m)) {
				return m
			}
		case <-timeout:
			c.t.Fatalf("no %s", action)
		}
	}
}

// update waits for a table update whose view passes cond.
func (c *client) update(cond func(table.View) bool) TableUpdate {
	c.t.Helper()
	var u TableUpdate
	c.wait(ActionTableUpdate, func(m msg) bool {
		u = TableUpdate{}
		m.decode(c.t, &u)
		return cond(u.View)
	})
	return u
}

func (c *client) errorMsg() ErrorOut {
	c.t.Helper()
	var e ErrorOut
	c.wait(ActionError, nil).decode(c.t, &e)
	return e
}

func createMsg(table, name string) string {
	return `{"action":"create-table","tablename":"` + table + `","username":"` + name + `"}`
}

func joinMsg(table, name string) string {
	return `{"action":"join-table","tablename":"` + table + `","username":"` + name + `"}`
}

func seatMsg(seat int) string {
	return `{"action":"take-seat","seat":` + strconv.Itoa(seat) + `,"amount":1000}`
}

// create makes a table as name and waits until it's made.
func create(t *testing.T, srv *httptest.Server, table, name string) *client {
	t.Helper()
	c := dial(t, srv)
	c.send(createMsg(table, name))
	c.wait(ActionWelcome, nil)
	return c
}

func join(t *testing.T, srv *httptest.Server, table, name string) *client {
	t.Helper()
	c := dial(t, srv)
	c.send(joinMsg(table, name))
	c.wait(ActionWelcome, nil)
	return c
}

// assertAlive checks a fresh client can still create a table.
func assertAlive(t *testing.T, srv *httptest.Server) {
	t.Helper()
	create(t, srv, "alive"+strconv.Itoa(time.Now().Nanosecond()), "probe")
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

func tableNames(h *Hub) []string {
	var names []string
	for _, a := range h.allTables() {
		names = append(names, a.name)
	}
	slices.Sort(names)
	return names
}

// waitForTables waits until exactly the wanted tables exist.
func waitForTables(t *testing.T, h *Hub, want ...string) {
	t.Helper()
	eventually(t, func() bool { return slices.Equal(tableNames(h), want) }, "tables = %v, want %v", tableNames(h), want)
}

func TestMalformedMessagesDoNotCrash(t *testing.T) {
	srv := newTestServer(t)
	c := dial(t, srv)
	c.send(
		// actions before joining a table
		`{"action":"send-message","message":"hi"}`,
		`{"action":"take-seat","seat":1,"amount":100}`,
		`{"action":"start-game"}`,
		`{"action":"leave-table","tablename":"nope"}`,
		`not json`,
		`{"action":"create-table","tablename":"x","username":"a","settings":"nope"}`,
		// actions at an empty table
		createMsg("t", "a"),
		`{"action":"start-game"}`,
		`{"action":"player-fold"}`,
		`{"action":"player-raise","to":-1}`,
		`{"action":"made-up"}`,
		// bad seats
		`{"action":"take-seat","seat":0,"amount":1000}`,
		`{"action":"take-seat","seat":99,"amount":1000}`,
		`{"action":"take-seat","seat":1,"amount":-1}`,
		seatMsg(1),
		seatMsg(1),
		// join or create without leaving
		joinMsg("other", "a"),
		createMsg("other", "a"),
	)
	c.wait(ActionJoined, func(m msg) bool { return strings.Contains(string(m.raw), `"other"`) })
	assertAlive(t, srv)
}

func TestConcurrentPlayers(t *testing.T) {
	srv := newTestServer(t)
	create(t, srv, "t", "host")

	var wg sync.WaitGroup
	for i := 1; i <= 4; i++ {
		c := dial(t, srv)
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.send(joinMsg("t", "p"+strconv.Itoa(i)), seatMsg(i))
			for range 2 { // stay under the rate limit
				c.send(
					`{"action":"start-game"}`,
					`{"action":"player-call"}`,
					`{"action":"player-check"}`,
					`{"action":"player-raise","to":80}`,
					`{"action":"player-fold"}`,
					`{"action":"send-message","message":"hi"}`,
				)
			}
		}()
	}
	wg.Wait()
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
	c := create(t, srv, "t", "p")
	for range 2 * maxMessagesPerSecond {
		c.conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"send-message","message":"spam"}`))
	}
	// the server should close the connection, so writes start failing
	eventually(t, func() bool { return c.conn.WriteMessage(websocket.TextMessage, []byte(`{}`)) != nil },
		"flooding client was not disconnected")
	assertAlive(t, srv)
}

func TestEmptyTablesClosed(t *testing.T) {
	emptied := testutil.ToFloat64(tablesClosed.WithLabelValues("empty"))
	srv, hub := newTestServerHub(t, newMemStore())
	c := create(t, srv, "a", "p")
	waitForTables(t, hub, "a")
	c.send(`{"action":"leave-table","tablename":"a"}`)
	waitForTables(t, hub)

	c.send(createMsg("b", "p"))
	waitForTables(t, hub, "b")
	c.send(seatMsg(1))
	c.update(func(v table.View) bool { return v.You == 1 })
	c.conn.Close()
	time.Sleep(100 * time.Millisecond)
	if !slices.Equal(tableNames(hub), []string{"b"}) {
		t.Fatal("a seated player's table waits for them to reconnect")
	}
	waitForTables(t, hub)
	if d := testutil.ToFloat64(tablesClosed.WithLabelValues("empty")) - emptied; d < 2 {
		t.Errorf("empty closes went up %v, want at least 2", d)
	}
}

func TestTableCap(t *testing.T) {
	srv, hub := newTestServerHub(t, newMemStore())
	for i := range maxTables + 5 {
		dial(t, srv).send(createMsg("t"+strconv.Itoa(i), "p"))
	}
	eventually(t, func() bool { return len(tableNames(hub)) >= maxTables }, "tables never reached the cap")
	time.Sleep(100 * time.Millisecond) // give extra creates time to (wrongly) succeed
	if n := len(tableNames(hub)); n != maxTables {
		t.Errorf("tables = %d, want %d", n, maxTables)
	}
}

func TestConnsPerIP(t *testing.T) {
	n := maxConnsPerIP
	t.Cleanup(func() { maxConnsPerIP = n }) // runs after the server closes
	maxConnsPerIP = 2                       // before the server starts, so its goroutines see it
	hits := testutil.ToFloat64(limitHits.WithLabelValues("conns_per_ip"))
	srv := newTestServer(t)
	first := dial(t, srv)
	dial(t, srv)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third dial: err = %v, resp = %v, want 429", err, resp)
	}
	if d := testutil.ToFloat64(limitHits.WithLabelValues("conns_per_ip")) - hits; d != 1 {
		t.Errorf("conns_per_ip hits went up %v, want 1", d)
	}

	first.conn.Close()
	eventually(t, func() bool {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err == nil {
			conn.Close()
		}
		return err == nil
	}, "closing a connection never freed its slot")
}

func TestTablesPerIP(t *testing.T) {
	defer func(n int) { maxTablesPerIP = n }(maxTablesPerIP)
	maxTablesPerIP = 2
	hub := newHub(newMemStore())
	create := func(name, ip string) error {
		_, err := hub.createTable(name, ip, "", table.DefaultSettings)
		return err
	}
	for _, name := range []string{"a", "b"} {
		if err := create(name, "1.1.1.1"); err != nil {
			t.Fatal(err)
		}
	}
	hits := testutil.ToFloat64(limitHits.WithLabelValues("tables_per_ip"))
	if err := create("c", "1.1.1.1"); err != errTooManyTablesIP {
		t.Errorf("third table: err = %v, want %v", err, errTooManyTablesIP)
	}
	if d := testutil.ToFloat64(limitHits.WithLabelValues("tables_per_ip")) - hits; d != 1 {
		t.Errorf("tables_per_ip hits went up %v, want 1", d)
	}
	if err := create("d", "2.2.2.2"); err != nil {
		t.Errorf("other address: %v", err)
	}
}

// TestPlay plays a hand over the wire, checking each player sees only their
// own cards.
func TestPlay(t *testing.T) {
	srv := newTestServer(t)
	a := create(t, srv, "t", "a")
	b := join(t, srv, "t", "b")
	w := join(t, srv, "t", "w")
	a.send(seatMsg(1))
	b.send(seatMsg(2))
	b.update(func(v table.View) bool { return len(v.Players) == 2 })
	a.send(`{"action":"start-game"}`)

	u := a.update(func(v table.View) bool { return v.Street == "preflop" })
	if u.View.Players[0].Cards[0] == 0 || u.View.Players[1].Cards[0] != 0 || !u.View.Legal.Turn {
		t.Fatalf("a's view %+v", u.View)
	}
	var dealt []string
	for _, e := range u.Events {
		dealt = append(dealt, e.Type)
		if e.Type == "dealt" && !strings.Contains(string(e.Data), `"seat":1`) {
			t.Fatalf("a got someone else's cards: %s", e.Data)
		}
	}
	if !slices.Contains(dealt, "dealt") || !slices.Contains(dealt, "hand-started") {
		t.Fatalf("events %v", dealt)
	}
	wu := w.update(func(v table.View) bool { return v.Street == "preflop" })
	if wu.View.Players[0].Cards[0] != 0 || slices.ContainsFunc(wu.Events, func(e EventEnvel) bool { return e.Type == "dealt" }) {
		t.Fatal("the watcher saw hole cards")
	}

	b.send(`{"action":"player-check"}`)
	if e := b.errorMsg(); e.Message != "not your turn" {
		t.Fatalf("error %+v", e)
	}
	a.send(`{"action":"player-raise","to":100}`)
	b.update(func(v table.View) bool { return v.Players[0].Bet == 100 && v.Legal.Turn })
	b.send(`{"action":"player-fold"}`)
	u = a.update(func(v table.View) bool { return v.Street == "" })
	if !slices.ContainsFunc(u.Events, func(e EventEnvel) bool { return e.Type == "hand-done" }) {
		t.Fatal("no hand-done event")
	}
	if u.View.Players[0].Stack != 1020 {
		t.Fatalf("a's stack %d", u.View.Players[0].Stack)
	}

	// between hands a shows their cards and b duck hunts
	a.send(`{"action":"show-cards"}`)
	w.update(func(v table.View) bool { return v.Players[0].Cards[0] != 0 })
	b.send(`{"action":"duck-hunt"}`)
	w.update(func(v table.View) bool { return len(v.Hunted) == 5 })

	a.send(`{"action":"get-hands","before":5}`)
	var h HandHistory
	a.wait(ActionHandHistory, nil).decode(t, &h)
	if len(h.Hands) != 1 || h.More {
		t.Fatalf("history %+v", h)
	}
}

func TestReconnect(t *testing.T) {
	srv, hub := newTestServerHub(t, newMemStore())
	host := create(t, srv, "t", "host")
	a := join(t, srv, "t", "a")
	a.send(seatMsg(1))
	var tok TokenOut
	a.wait(ActionToken, nil).decode(t, &tok)
	a.conn.Close()
	host.update(func(v table.View) bool { return len(v.Players) == 1 && v.Players[0].Disconnected })

	back := dial(t, srv)
	back.send(`{"action":"join-table","tablename":"t","username":"anyone","token":"` + tok.Token + `"}`)
	var j Joined
	back.wait(ActionJoined, nil).decode(t, &j)
	if j.Username != "a" {
		t.Fatalf("rejoined as %q", j.Username)
	}
	host.update(func(v table.View) bool { return len(v.Players) == 1 && !v.Players[0].Disconnected })

	// a second tab with the token takes over, closing the first
	tab := dial(t, srv)
	tab.send(`{"action":"join-table","tablename":"t","username":"anyone","token":"` + tok.Token + `"}`)
	tab.wait(ActionWelcome, nil)
	eventually(t, func() bool { return back.conn.WriteMessage(websocket.TextMessage, []byte(`{}`)) != nil },
		"the old tab stayed open")
	tab.send(`{"action":"send-message","message":"still seated"}`)
	host.wait(ActionChat, func(m msg) bool { return strings.Contains(string(m.raw), "still seated") })
	time.Sleep(2 * timing.ReconnectGrace)
	if hub.table("t").summary.Load().Players != 1 {
		// the old tab's disconnect mustn't start a grace period
		t.Fatal("a lost their seat")
	}
}

func TestPassword(t *testing.T) {
	srv := newTestServer(t)
	host := dial(t, srv)
	host.send(`{"action":"create-table","tablename":"t","username":"host","password":"pw"}`)
	var j Joined
	host.wait(ActionJoined, nil).decode(t, &j)
	if j.Password != "pw" {
		t.Fatal("the creator gets the password for invites")
	}
	c := dial(t, srv)
	c.send(joinMsg("t", "a"))
	if e := c.errorMsg(); e.Code != "password-required" {
		t.Fatalf("got %+v", e)
	}
	c.send(`{"action":"join-table","tablename":"t","username":"a","password":"pw"}`)
	c.wait(ActionWelcome, nil)
}

func TestTablesList(t *testing.T) {
	srv := newTestServer(t)
	create(t, srv, "private", "a")
	c := dial(t, srv)
	c.send(`{"action":"create-table","tablename":"public","username":"b","settings":{"public":true}}`)
	c.wait(ActionWelcome, nil)
	c.send(`{"action":"list-tables"}`)
	var got Tables
	c.wait(ActionTables, nil).decode(t, &got)
	if len(got.Tables) != 1 || got.Tables[0].Name != "public" || got.Tables[0].BigBlind != table.DefaultSettings.BigBlind {
		t.Fatalf("tables %+v", got.Tables)
	}
}

func TestUpdateSettings(t *testing.T) {
	srv := newTestServer(t)
	host := create(t, srv, "t", "host")
	a := join(t, srv, "t", "a")

	// the rest are zero, so the buy-ins don't fit the blinds
	host.send(`{"action":"update-settings","settings":{"bb":20}}`)
	if e := host.errorMsg(); !strings.Contains(e.Message, "buy-in") {
		t.Fatalf("partial settings: %+v", e)
	}
	set := table.DefaultSettings
	set.SmallBlind, set.BigBlind, set.Public = 25, 50, true
	b, _ := json.Marshal(set)
	a.send(`{"action":"update-settings","settings":` + string(b) + `}`)
	if e := a.errorMsg(); e.Message != "only the host can do that" {
		t.Fatalf("not host: %+v", e)
	}
	host.send(`{"action":"update-settings","settings":` + string(b) + `}`)
	a.update(func(v table.View) bool { return v.Settings.BigBlind == 50 })

	// the table is listed once it's public
	a.send(`{"action":"list-tables"}`)
	var got Tables
	a.wait(ActionTables, nil).decode(t, &got)
	if len(got.Tables) != 1 || got.Tables[0].BigBlind != 50 {
		t.Fatalf("tables %+v", got.Tables)
	}

	host.send(`{"action":"set-password","password":"pw"}`)
	var pw PasswordOut
	a.wait(ActionPassword, nil).decode(t, &pw)
	if pw.Password != "pw" {
		t.Fatalf("a got %+v", pw)
	}
	c := dial(t, srv)
	c.send(joinMsg("t", "c"))
	if e := c.errorMsg(); e.Code != "password-required" {
		t.Fatalf("got %+v", e)
	}
}

func TestEndedTableCloses(t *testing.T) {
	srv, hub := newTestServerHub(t, newMemStore())
	c := create(t, srv, "t", "host")
	c.send(`{"action":"end-session"}`)
	c.update(func(v table.View) bool { return v.Ended })
	if e := c.errorMsg(); e.Code != table.CodeClosed {
		t.Fatalf("got %+v", e)
	}
	waitForTables(t, hub)
}

// openTables names the tables the store would bring back.
func openTables(t *testing.T, store Store) []string {
	t.Helper()
	saved, err := store.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range saved {
		names = append(names, s.Name)
	}
	slices.Sort(names)
	return names
}

func TestShutdown(t *testing.T) {
	store := newMemStore()
	srv, hub := newTestServerHub(t, store)
	c := create(t, srv, "t", "host")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hub.shutdown(ctx)
	if e := c.errorMsg(); e.Code != table.CodeRestarting || e.Message != "the server is restarting" {
		t.Fatalf("got %+v", e)
	}
	if got := openTables(t, store); !slices.Equal(got, []string{"t"}) {
		t.Fatalf("a shutdown keeps the table open in the store, got %v", got)
	}
}

func TestClosedTablesKept(t *testing.T) {
	store := newMemStore()
	srv, hub := newTestServerHub(t, store)
	c := create(t, srv, "t", "host")
	c.send(`{"action":"leave-table"}`)
	waitForTables(t, hub)
	if got := openTables(t, store); len(got) != 0 {
		t.Fatalf("open tables %v", got)
	}
	// a new table can take the name
	create(t, srv, "t", "host")
	eventually(t, func() bool { return len(openTables(t, store)) == 1 }, "the new table isn't saved")

	if n, _ := store.Prune(context.Background(), time.Now().Add(-time.Hour)); n != 0 {
		t.Fatalf("pruned %d tables closed within retention", n)
	}
	if n, _ := store.Prune(context.Background(), time.Now()); n != 1 {
		t.Fatalf("pruned %d tables, want the closed one", n)
	}
}

// TestRestore brings back a table with a hand under way after a restart.
func TestRestore(t *testing.T) {
	store := newMemStore()
	srv, hub := newTestServerHub(t, store)
	host := create(t, srv, "t", "host")
	host.send(seatMsg(1),
		`{"action":"add-bot","seat":2,"amount":1000,"name":"bob","level":"easy"}`,
		`{"action":"start-game"}`)
	var tok TokenOut
	host.wait(ActionToken, nil).decode(t, &tok)
	host.update(func(v table.View) bool { return v.Betting && v.ToAct == 1 })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hub.shutdown(ctx)

	srv2, hub2 := newTestServerHub(t, store)
	if err := hub2.restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForTables(t, hub2, "t")
	back := dial(t, srv2)
	back.send(`{"action":"join-table","tablename":"t","username":"x","token":"` + tok.Token + `"}`)
	var w Welcome
	back.wait(ActionWelcome, nil).decode(t, &w)
	if w.View.You != 1 || !w.View.Legal.Turn || w.View.Players[0].Cards[0] == 0 {
		t.Fatalf("restored view %+v", w.View)
	}
	back.send(`{"action":"player-fold"}`)
	back.update(func(v table.View) bool { return v.Street == "" && v.Hand == 1 })
}

// TestLateShowSaved saves a hand's record again when cards are shown after it.
// On SQLite, since the memory store shares the records the table changes.
func TestLateShowSaved(t *testing.T) {
	store := openTestStore(t, "sqlite:"+filepath.Join(t.TempDir(), "poker.db"))
	srv, _ := newTestServerHub(t, store)
	a := create(t, srv, "t", "a")
	b := join(t, srv, "t", "b")
	a.send(seatMsg(1))
	b.send(seatMsg(2))
	b.update(func(v table.View) bool { return len(v.Players) == 2 })
	a.send(`{"action":"start-game"}`)
	a.update(func(v table.View) bool { return v.Legal.Turn })
	a.send(`{"action":"player-fold"}`)
	b.update(func(v table.View) bool { return v.Street == "" && v.Hand == 1 })
	b.send(`{"action":"show-cards"}`)
	a.update(func(v table.View) bool { return v.Players[1].Shown })

	eventually(t, func() bool {
		saved, err := store.Open(context.Background())
		return err == nil && len(saved) == 1 && len(saved[0].Hands) == 1 && saved[0].Hands[0].Record.Players[1].Shown
	}, "the shown cards weren't saved")
}

// crashStore stops saving once cut, like a server that died.
type crashStore struct {
	Store
	cut atomic.Bool
}

func (c *crashStore) Save(ctx context.Context, cp Checkpoint) error {
	if c.cut.Load() {
		return nil
	}
	return c.Store.Save(ctx, cp)
}

// TestCrashRestore brings back a table that died mid-hand from before the
// hand, which is dealt again the same.
func TestCrashRestore(t *testing.T) {
	t.Run("memory", func(t *testing.T) { testCrashRestore(t, newMemStore()) })
	t.Run("sqlite", func(t *testing.T) {
		testCrashRestore(t, openTestStore(t, "sqlite:"+filepath.Join(t.TempDir(), "poker.db")))
	})
}

func testCrashRestore(t *testing.T, saved Store) {
	store := &crashStore{Store: saved}
	srv, _ := newTestServerHub(t, store)
	host := dial(t, srv)
	host.timeout = 5 * time.Second // the next hand waits NextHandDelay
	host.send(`{"action":"create-table","tablename":"t","username":"host","settings":{"nextHandDelay":2000}}`)
	host.wait(ActionWelcome, nil)
	host.send(seatMsg(1),
		`{"action":"add-bot","seat":2,"amount":1000,"name":"bob","level":"easy"}`,
		`{"action":"start-game"}`)
	var tok TokenOut
	host.wait(ActionToken, nil).decode(t, &tok)
	// fold through the first hand, into the second
	u := host.update(func(v table.View) bool {
		if v.Hand == 2 && v.Street != "" {
			return true
		}
		if v.Legal.Turn {
			host.send(`{"action":"player-fold"}`)
		}
		return false
	})
	cards := u.View.Players[0].Cards
	store.cut.Store(true)
	open, _ := store.Open(context.Background())
	var s table.State
	if json.Unmarshal(open[0].State, &s); s.Game.InHand() || s.Game.Hands != 1 {
		t.Fatalf("saved mid-hand, at hand %d", s.Game.Hands)
	}

	srv2, hub2 := newTestServerHub(t, store.Store)
	if err := hub2.restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForTables(t, hub2, "t")
	back := dial(t, srv2)
	back.timeout = 5 * time.Second
	back.send(`{"action":"join-table","tablename":"t","username":"x","token":"` + tok.Token + `"}`)
	var w Welcome
	back.wait(ActionWelcome, nil).decode(t, &w)
	if w.View.You != 1 || len(w.History) != 1 {
		t.Fatalf("restored as seat %d with %d hands of history, want 1 of each", w.View.You, len(w.History))
	}
	// the deal timer was due before the restart, so hand 2 may already be out
	v := w.View
	if v.Hand != 2 || v.Street == "" {
		v = back.update(func(v table.View) bool { return v.Hand == 2 && v.Street != "" }).View
	}
	if !slices.Equal(v.Players[0].Cards, cards) {
		t.Fatalf("hand 2 dealt %v, then %v", cards, v.Players[0].Cards)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "10.0.0.1:5000"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 1.2.3.4")
	r.Header.Set("X-Real-IP", "6.6.6.6")
	if ip := clientIP(r, false); ip != "10.0.0.1" {
		t.Errorf("untrusted: got %q, want the socket address", ip)
	}
	if ip := clientIP(r, true); ip != "1.2.3.4" {
		t.Errorf("trusted: got %q, want the entry the proxy added", ip)
	}
}
