package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/poker"
	"github.com/gorilla/websocket"
)

// short timers so tests run fast
const (
	nextHandDelay = 50 * time.Millisecond
	turnTimeout   = 500 * time.Millisecond
)

func TestMain(m *testing.M) {
	defaultSettings.NextHandDelay = uint(nextHandDelay.Milliseconds())
	defaultSettings.TurnTime = uint(turnTimeout.Milliseconds())
	defaultSettings.MinBuy = 100
	defaultSettings.MaxBuy = 10000
	reconnectGrace = 300 * time.Millisecond
	botDelay = func() time.Duration { return 10 * time.Millisecond }
	endedTableTTL = 500 * time.Millisecond
	os.Exit(m.Run())
}

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

func createMsg(table, name string) string {
	return `{"action":"create-table","tablename":"` + table + `","username":"` + name + `"}`
}

func joinMsg(table, name string) string {
	return `{"action":"join-table","tablename":"` + table + `","username":"` + name + `"}`
}

func seatMsg(seat int) string {
	return `{"action":"take-seat","seatID":` + strconv.Itoa(seat) + `,"buyIn":100}`
}

// createAndWait creates a table and waits until the server has made it, so
// other clients can join.
func createAndWait(t *testing.T, srv *httptest.Server, table, name string) (*websocket.Conn, received) {
	t.Helper()
	conn, r := dialRecv(t, srv)
	send(t, conn, createMsg(table, name))
	waitGame(t, r.games, func(poker.GameView) bool { return true })
	return conn, r
}

// assertAlive checks a fresh client can still join and get a game update.
func assertAlive(t *testing.T, srv *httptest.Server) {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("server not accepting connections: %s", err)
	}
	defer conn.Close()
	send(t, conn, createMsg("alive", "probe"))
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
		createMsg("t", "a"),
		`{"action":"start-game"}`,
		`{"action":"player-fold"}`,
		`{"action":"player-call"}`,
		// bad seats
		`{"action":"take-seat","username":"a","seatID":0,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":99,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":1,"buyIn":100}`,
		`{"action":"take-seat","username":"a","seatID":1,"buyIn":100}`,
		// join or create without leaving
		joinMsg("other", "a"),
		createMsg("other", "a"),
	)
	time.Sleep(200 * time.Millisecond)
	assertAlive(t, srv)
}

func TestConcurrentPlayers(t *testing.T) {
	srv := newTestServer(t)
	createAndWait(t, srv, "t", "host")

	var wg sync.WaitGroup
	for i := 1; i <= 4; i++ {
		conn := dial(t, srv)
		seat := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			send(t, conn, joinMsg("t", "p"+strconv.Itoa(seat)), seatMsg(seat))
			for j := 0; j < 2; j++ { // stay under the rate limit
				send(t, conn,
					`{"action":"start-game"}`,
					`{"action":"player-call"}`,
					`{"action":"player-check"}`,
					`{"action":"player-raise","amount":40}`,
					`{"action":"player-fold"}`,
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
	a, _ := createAndWait(t, srv, "t", "a")
	b := dial(t, srv)
	send(t, a, seatMsg(1))
	send(t, b, joinMsg("t", "b"), seatMsg(2))
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
	send(t, conn, createMsg("t", "p"))
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

	send(t, conn, createMsg("a", "p"))
	waitForTables(t, hub, "a")

	send(t, conn, `{"action":"leave-table","tablename":"a"}`)
	waitForTables(t, hub)

	send(t, conn, createMsg("c", "p"))
	waitForTables(t, hub, "c")
	conn.Close()
	waitForTables(t, hub)
}

func TestTableCap(t *testing.T) {
	srv, hub := newTestServerHub(t)
	for i := 0; i < maxTables+5; i++ {
		conn := dial(t, srv)
		send(t, conn, createMsg("t"+strconv.Itoa(i), "p"))
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
	host, _ := createAndWait(t, srv, "t", "p1")
	send(t, host, seatMsg(1))
	send(t, dial(t, srv), joinMsg("t", "p2"), seatMsg(2))
	spectator := dial(t, srv)
	send(t, spectator, joinMsg("t", "s"), `{"action":"start-game"}`)
	time.Sleep(200 * time.Millisecond)
	if isRunning(t, hub, "t") {
		t.Fatal("spectator started the game")
	}

	player := dial(t, srv)
	send(t, player, joinMsg("t", "p3"), seatMsg(3), `{"action":"start-game"}`)
	eventually(t, func() bool { return isRunning(t, hub, "t") }, "seated player could not start the game")
}

// received collects what the server sends one client.
type received struct {
	games   chan poker.GameView
	hosts   chan string        // host name of each game update
	ends    chan int64         // turnEndsIn of each game update
	sitOut  chan bool          // sitOutNext of each game update
	watch   chan []string      // watchers of each game update
	ledger  chan ledgerView    // ledger of each game update
	away    chan []string      // disconnected of each game update
	session chan session       // session state of each game update
	sets    chan tableSettings // settings of each game update
	errors  chan string
	codes   chan string // code of each error
	locked  chan bool   // hasPassword of each game update
	joined  chan string // password of each joined-table
	logs    chan string
	tokens  chan string // session tokens from taking a seat
	chat    chan string // chat messages, as "name: message"
	stats   chan updateStats
	bots    chan map[string]string // bots of each game update
}

// dialRecv is dial, but also collects the game views and errors the client receives.
func dialRecv(t *testing.T, srv *httptest.Server) (*websocket.Conn, received) {
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	r := received{
		games:   make(chan poker.GameView, 256),
		hosts:   make(chan string, 256),
		ends:    make(chan int64, 256),
		sitOut:  make(chan bool, 256),
		watch:   make(chan []string, 256),
		ledger:  make(chan ledgerView, 256),
		away:    make(chan []string, 256),
		session: make(chan session, 256),
		stats:   make(chan updateStats, 256),
		sets:    make(chan tableSettings, 256),
		errors:  make(chan string, 256),
		codes:   make(chan string, 256),
		locked:  make(chan bool, 256),
		joined:  make(chan string, 256),
		logs:    make(chan string, 256),
		tokens:  make(chan string, 256),
		chat:    make(chan string, 256),
		bots:    make(chan map[string]string, 256),
	}
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var u struct {
				Action  string            `json:"action"`
				Game    poker.GameView    `json:"game"`
				Message string            `json:"message"`
				Host    string            `json:"host"`
				Token   string            `json:"token"`
				Ends    int64             `json:"turnEndsIn"`
				SitOut  bool              `json:"sitOutNext"`
				Watch   []string          `json:"watchers"`
				Ledger  ledgerView        `json:"ledger"`
				Away    []string          `json:"disconnected"`
				Name    string            `json:"username"`
				Sets    tableSettings     `json:"settings"`
				Code    string            `json:"code"`
				Locked  bool              `json:"hasPassword"`
				Pass    string            `json:"password"`
				Paused  bool              `json:"paused"`
				Ending  bool              `json:"ending"`
				Ended   bool              `json:"ended"`
				Hands   []handStats       `json:"hands"`
				Full    bool              `json:"full"`
				Bots    map[string]string `json:"bots"`
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
				select {
				case r.hosts <- u.Host:
				default:
				}
				select {
				case r.ends <- u.Ends:
				default:
				}
				select {
				case r.sitOut <- u.SitOut:
				default:
				}
				select {
				case r.watch <- u.Watch:
				default:
				}
				select {
				case r.ledger <- u.Ledger:
				default:
				}
				select {
				case r.away <- u.Away:
				default:
				}
				select {
				case r.session <- session{u.Paused, u.Ending, u.Ended}:
				default:
				}
				select {
				case r.sets <- u.Sets:
				default:
				}
				select {
				case r.locked <- u.Locked:
				default:
				}
				select {
				case r.bots <- u.Bots:
				default:
				}
			case actionError:
				select {
				case r.errors <- u.Message:
				default:
				}
				select {
				case r.codes <- u.Code:
				default:
				}
			case actionUpdateStats:
				select {
				case r.stats <- updateStats{Hands: u.Hands, Full: u.Full}:
				default:
				}
			case actionJoinedTable:
				select {
				case r.joined <- u.Pass:
				default:
				}
			case actionUpdatePlayerUUID:
				select {
				case r.tokens <- u.Token:
				default:
				}
			case actionNewLog:
				select {
				case r.logs <- u.Message:
				default:
				}
			case actionNewMessage:
				select {
				case r.chat <- u.Name + ": " + u.Message:
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
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	spectator, sRecv := dialRecv(t, srv)
	send(t, a, seatMsg(1))
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	send(t, spectator, joinMsg("t", "s"))
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
	a, aRecv, b, bRecv := seatTwo(t, srv)
	spectator, sRecv := dialRecv(t, srv)
	send(t, spectator, joinMsg("t", "s"))

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
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.ActionNum == 1 })
	if !g.Players[1].In {
		t.Error("b was folded by someone else")
	}
}

func TestSeatErrors(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"))
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })

	send(t, a, seatMsg(2))
	if msg := waitError(t, aRecv); msg != "you already have a seat" {
		t.Errorf("second seat: got %q", msg)
	}
	send(t, b, seatMsg(1))
	if msg := waitError(t, bRecv); msg != "seat taken" {
		t.Errorf("taken seat: got %q", msg)
	}
	send(t, b, seatMsg(9))
	if msg := waitError(t, bRecv); msg != "invalid seat" {
		t.Errorf("seat past the last: got %q", msg)
	}
	send(t, b, `{"action":"take-seat","seatID":2,"buyIn":999999}`)
	if msg := waitError(t, bRecv); !strings.HasPrefix(msg, "buy-in must be") {
		t.Errorf("bad buy-in: got %q", msg)
	}
	send(t, b, seatMsg(8))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
}

func TestCreateAndJoinErrors(t *testing.T) {
	srv := newTestServer(t)
	createAndWait(t, srv, "t", "alice")

	tests := []struct {
		msg  string
		want string
	}{
		{createMsg("t", "bob"), errTableExists.Error()},
		{joinMsg("missing", "bob"), errNoTable.Error()},
		{joinMsg("t", "ALICE"), errNameTaken.Error()},
		{joinMsg("t", "System"), "invalid name"},
	}
	for _, tt := range tests {
		conn, r := dialRecv(t, srv)
		send(t, conn, tt.msg)
		if got := waitError(t, r); got != tt.want {
			t.Errorf("%s: got error %q, want %q", tt.msg, got, tt.want)
		}
	}
}

func TestServerDealsNextHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)

	// heads up, a acts first; folding ends the hand
	send(t, a, `{"action":"player-fold"}`)
	time.Sleep(5 * nextHandDelay)

	var logs []string
	for len(aRecv.logs) > 0 {
		logs = append(logs, <-aRecv.logs)
	}
	count := func(prefix string) int {
		n := 0
		for _, l := range logs {
			if strings.HasPrefix(l, prefix) {
				n++
			}
		}
		return n
	}
	if n := count("a folds"); n != 1 {
		t.Errorf("fold logged %d times, logs: %q", n, logs)
	}
	if n := count("b wins 30"); n != 1 {
		t.Errorf("win logged %d times, logs: %q", n, logs)
	}
	if n := count("starting new hand"); n != 2 {
		t.Errorf("%d hands dealt, want 2 (start and next), logs: %q", n, logs)
	}
}

// collectLogs returns the logs r receives over d.
func collectLogs(r received, d time.Duration) []string {
	var logs []string
	timeout := time.After(d)
	for {
		select {
		case l := <-r.logs:
			logs = append(logs, l)
		case <-timeout:
			return logs
		}
	}
}

// seatTwo seats a in seat 1, then b in seat 2, and starts the game. Seating
// in order makes a the dealer, who acts first heads up.
func seatTwo(t *testing.T, srv *httptest.Server) (a *websocket.Conn, aRecv received, b *websocket.Conn, bRecv received) {
	t.Helper()
	a, aRecv = createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	b, bRecv = dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, a, `{"action":"start-game"}`)
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting })
	if actor := g.Players[g.ActionNum].Username; actor != "a" {
		t.Fatalf("%s acts first, want a", actor)
	}
	return a, aRecv, b, bRecv
}

func TestTurnTimeout(t *testing.T) {
	srv := newTestServer(t)
	_, aRecv, _, _ := seatTwo(t, srv)

	// a is small blind facing the big blind, so is folded
	logs := collectLogs(aRecv, turnTimeout+300*time.Millisecond)
	if !slices.Contains(logs, "a folds (out of time)") {
		t.Errorf("a was not folded, logs: %q", logs)
	}
}

func TestDisconnectFoldsAndRemoves(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, _ := seatTwo(t, srv)

	// b leaves while it's a's turn; once a calls, b is folded and removed
	b.Close()
	time.Sleep(100 * time.Millisecond)
	send(t, a, `{"action":"player-call"}`)
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	if g.Players[0].Username != "a" {
		t.Errorf("wrong player removed: %+v", g.Players)
	}
	logs := collectLogs(aRecv, 100*time.Millisecond)
	for _, want := range []string{"b folds", "a wins 40", "b left the table"} {
		if !slices.Contains(logs, want) {
			t.Errorf("missing log %q, logs: %q", want, logs)
		}
	}
}

func TestLeaveBetweenHands(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b := dial(t, srv)
	send(t, a, seatMsg(1))
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, b, `{"action":"leave-table","tablename":"t"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
}

func waitToken(t *testing.T, r received) string {
	t.Helper()
	select {
	case token := <-r.tokens:
		return token
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for session token")
		return ""
	}
}

func TestReconnectKeepsSeat(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	token := waitToken(t, bRecv)
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })

	// b drops and comes back on a new connection within the grace period,
	// reclaiming its seat and name even with a different name in the message
	b.Close()
	b2, b2Recv := dialRecv(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"whoever","token":"`+token+`"}`)
	waitGame(t, b2Recv.games, func(g poker.GameView) bool { return true })
	time.Sleep(2 * reconnectGrace)
	send(t, a, `{"action":"start-game"}`)
	g := waitGame(t, b2Recv.games, func(g poker.GameView) bool { return g.Betting })
	if len(g.Players) != 2 {
		t.Fatalf("players = %d, want 2", len(g.Players))
	}
	for _, p := range g.Players {
		if p.Username == "b" && p.Cards[0] == 0 {
			t.Error("reconnected b can't see its own cards")
		}
	}
}

func TestReconnectLogged(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	token := waitToken(t, bRecv)
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })

	b.Close()
	waitLog(t, aRecv, "b disconnected")
	for len(aRecv.chat) > 0 {
		<-aRecv.chat // b's first join
	}
	b2 := dial(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"b","token":"`+token+`"}`)
	waitLog(t, aRecv, "b reconnected")
	// the reconnect is logged, not announced again in chat
	time.Sleep(200 * time.Millisecond)
	for len(aRecv.chat) > 0 {
		if m := <-aRecv.chat; m == gameAdminName+": b has joined" {
			t.Error("reconnect announced as a join")
		}
	}
}

func TestWatchers(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	b := dial(t, srv)
	send(t, b, joinMsg("t", "b"))
	waitWatching(t, aRecv, "b")

	send(t, b, seatMsg(2))
	waitWatching(t, aRecv)

	send(t, b, `{"action":"leave-seat"}`)
	waitWatching(t, aRecv, "b")

	b.Close()
	waitWatching(t, aRecv)
}

func waitWatching(t *testing.T, r received, names ...string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case w := <-r.watch:
			if slices.Equal(w, names) || len(w) == 0 && len(names) == 0 {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for watchers %q", names)
		}
	}
}

func TestReconnectAfterGraceFails(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	token := waitToken(t, bRecv)
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })

	b.Close()
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })

	// the token no longer works, so this is a normal join as a spectator
	b2, b2Recv := dialRecv(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"b","token":"`+token+`"}`)
	g := waitGame(t, b2Recv.games, func(g poker.GameView) bool { return true })
	if len(g.Players) != 1 {
		t.Errorf("players = %d, want 1", len(g.Players))
	}
}

func TestTableKeptForReconnect(t *testing.T) {
	srv, hub := newTestServerHub(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	token := waitToken(t, aRecv)

	// the only player drops: the table waits for them, then goes
	a.Close()
	time.Sleep(reconnectGrace / 3)
	if names := tableNames(hub); !slices.Equal(names, []string{"t"}) {
		t.Fatalf("tables = %v, want [t] during grace period", names)
	}
	a2, a2Recv := dialRecv(t, srv)
	send(t, a2, `{"action":"join-table","tablename":"t","username":"a","token":"`+token+`"}`)
	waitGame(t, a2Recv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	a2.Close()
	waitForTables(t, hub)
}

func TestTurnEndsIn(t *testing.T) {
	srv := newTestServer(t)
	a, _, _, _ := seatTwo(t, srv)
	s, sRecv := dialRecv(t, srv)
	send(t, s, joinMsg("t", "s"))

	// a fresh client's games and ends are filled together, so read them in
	// pairs. The first is s's view on joining, mid-hand.
	sawBetting, sawBetween := false, false
	for !sawBetting || !sawBetween {
		g := waitGame(t, sRecv.games, func(poker.GameView) bool { return true })
		ends := <-sRecv.ends
		if !sawBetting {
			send(t, a, `{"action":"player-fold"}`)
		}
		switch {
		case g.Betting:
			sawBetting = true
			if ends <= 0 || ends > turnTimeout.Milliseconds() {
				t.Errorf("turnEndsIn = %d while betting, want in (0, %d]", ends, turnTimeout.Milliseconds())
			}
		default:
			sawBetween = sawBetween || g.Running
			if ends != 0 {
				t.Errorf("turnEndsIn = %d between hands, want 0", ends)
			}
		}
	}
}

// waitLog waits for the client to receive log msg.
func waitLog(t *testing.T, r received, msg string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case l := <-r.logs:
			if l == msg {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for log %q", msg)
		}
	}
}

// byName returns the player named name in g.
func byName(t *testing.T, g poker.GameView, name string) (in, ready bool) {
	t.Helper()
	for _, p := range g.Players {
		if p.Username == name {
			return p.In, p.Ready
		}
	}
	t.Fatalf("no player %q", name)
	return false, false
}

func TestLeaveSeatBetweenHands(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, bRecv := dialRecv(t, srv)
	send(t, a, seatMsg(1))
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitToken(t, bRecv)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })

	send(t, b, `{"action":"leave-seat"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	if token := waitToken(t, bRecv); token != "" {
		t.Errorf("spectator got a session token %q", token)
	}
	send(t, b, `{"action":"leave-seat"}`)
	if msg := waitError(t, bRecv); msg != "you don't have a seat" {
		t.Errorf("second leave-seat: got %q", msg)
	}

	// b's name stays taken while b is connected
	other, otherRecv := dialRecv(t, srv)
	send(t, other, joinMsg("t", "b"))
	if msg := waitError(t, otherRecv); msg != errNameTaken.Error() {
		t.Errorf("joining as b: got %q", msg)
	}

	// b now watches a new hand as a spectator
	c := dial(t, srv)
	send(t, c, joinMsg("t", "c"), seatMsg(3))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, a, `{"action":"start-game"}`)
	g := waitGame(t, bRecv.games, func(g poker.GameView) bool { return g.Betting })
	for _, p := range g.Players {
		if p.Cards[0] != 0 {
			t.Errorf("spectator b sees %s's cards", p.Username)
		}
	}

	n := 0
	for _, l := range collectLogs(aRecv, 100*time.Millisecond) {
		if l == "b left the table" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("leave logged %d times, want 1", n)
	}
}

func TestLeaveSeatOnTurnFolds(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, bRecv := seatTwo(t, srv)

	send(t, a, `{"action":"leave-seat"}`)
	waitGame(t, bRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	logs := collectLogs(bRecv, 100*time.Millisecond)
	count := map[string]int{}
	for _, l := range logs {
		count[l]++
	}
	for _, want := range []string{"a folds", "b wins 30", "a left the table"} {
		if count[want] != 1 {
			t.Errorf("log %q seen %d times, logs: %q", want, count[want], logs)
		}
	}
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	if g.Players[0].Username != "b" {
		t.Errorf("wrong player left: %+v", g.Players)
	}
}

// seatThree seats a, b and c in seats 1 to 3 without starting.
func seatThree(t *testing.T, srv *httptest.Server) (conns map[string]*websocket.Conn, aRecv received) {
	t.Helper()
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	conns = map[string]*websocket.Conn{"a": a}
	for i, name := range []string{"b", "c"} {
		waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == i+1 })
		conns[name] = dial(t, srv)
		send(t, conns[name], joinMsg("t", name), seatMsg(i+2))
	}
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 3 })
	return conns, aRecv
}

func TestSitOutBetweenHands(t *testing.T) {
	srv := newTestServer(t)
	conns, aRecv := seatThree(t, srv)

	send(t, conns["a"], `{"action":"sit-out"}`)
	waitLog(t, aRecv, "a sits out")
	send(t, conns["b"], `{"action":"start-game"}`)
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting })
	if in, _ := byName(t, g, "a"); in {
		t.Fatal("a was dealt in while sitting out")
	}

	// a comes back, and is dealt into the next hand
	send(t, conns["a"], `{"action":"sit-out"}`)
	waitLog(t, aRecv, "a is back")
	actor := g.Players[g.ActionNum].Username
	send(t, conns[actor], `{"action":"player-fold"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool {
		in, _ := byName(t, g, "a")
		return g.Betting && in
	})
}

func TestSitOutMidHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, _ := seatTwo(t, srv)

	send(t, a, `{"action":"sit-out"}`)
	waitLog(t, aRecv, "a will sit out the next hand")
	// b changes their mind before the hand ends
	send(t, b, `{"action":"sit-out"}`)
	waitLog(t, aRecv, "b will sit out the next hand")
	send(t, b, `{"action":"sit-out"}`)
	waitLog(t, aRecv, "b will play the next hand")

	send(t, a, `{"action":"player-fold"}`)
	waitLog(t, aRecv, "a sits out")
	g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return !g.Betting })
	if _, ready := byName(t, g, "a"); ready {
		t.Error("a still ready after the hand")
	}
	if _, ready := byName(t, g, "b"); !ready {
		t.Error("b sat out after cancelling")
	}

	// sitting back in deals the next hand with a in it
	time.Sleep(3 * nextHandDelay)
	send(t, a, `{"action":"sit-out"}`)
	waitLog(t, aRecv, "a is back")
	waitGame(t, aRecv.games, func(g poker.GameView) bool {
		in, _ := byName(t, g, "a")
		return g.Betting && in
	})
}

// nextSitOut returns the sitOutNext of the next game update r gets.
// waitSitOut waits for a game update with sitOutNext set to want. Earlier
// updates may still be arriving, so it skips any that don't match.
func waitSitOut(t *testing.T, r received, want bool, msg string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case v := <-r.sitOut:
			if v == want {
				return
			}
		case <-timeout:
			t.Error(msg)
			return
		}
	}
}

func TestSitOutNextConfirmed(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, bRecv := seatTwo(t, srv)
	send(t, a, `{"action":"sit-out"}`)
	waitSitOut(t, aRecv, true, "a never got sitOutNext after sitting out mid-hand")
	// b didn't ask, so none of b's updates say so
	timeout := time.After(100 * time.Millisecond)
	for done := false; !done; {
		select {
		case v := <-bRecv.sitOut:
			if v {
				t.Error("b got sitOutNext true without asking")
			}
		case <-timeout:
			done = true
		}
	}

	// cancelling clears it
	send(t, a, `{"action":"sit-out"}`)
	waitSitOut(t, aRecv, false, "a's sitOutNext stayed set after cancelling")
}

func TestRebuyErrors(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, `{"action":"rebuy","amount":100}`)
	if msg := waitError(t, aRecv); msg != "take a seat first" {
		t.Errorf("unseated rebuy: got %q", msg)
	}
	send(t, a, seatMsg(1))
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	send(t, a, `{"action":"rebuy","amount":0}`)
	if msg := waitError(t, aRecv); msg != "rebuy amount must be more than 0" {
		t.Errorf("zero rebuy: got %q", msg)
	}
	send(t, a, `{"action":"rebuy","amount":9901}`)
	if msg := waitError(t, aRecv); msg != "rebuy can't take your stack above 10000" {
		t.Errorf("rebuy above max: got %q", msg)
	}
	send(t, a, `{"action":"rebuy","amount":9900}`)
	waitLog(t, aRecv, "a rebuys for 9900")
}

func TestRebuyMidHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)
	send(t, a, `{"action":"rebuy","amount":100}`)
	if msg := waitError(t, aRecv); msg != "can't rebuy during a hand" {
		t.Errorf("mid-hand rebuy: got %q", msg)
	}
}

func TestRebuyDealsBustedPlayerIn(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, b, bRecv := seatTwo(t, srv)
	conns := map[string]*websocket.Conn{"a": a, "b": b}
	recvs := map[string]received{"a": aRecv, "b": bRecv}

	// go all in each hand until someone busts; a split pot just deals again.
	// a (player 0) acts first, with 90 left after the small blind.
	first, pusher, stack := uint(0), "a", uint(90)
	busted := ""
	for {
		send(t, conns[pusher], `{"action":"player-raise","amount":`+strconv.Itoa(int(stack))+`}`)
		g := waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting && g.ActionNum != first })
		send(t, conns[g.Players[g.ActionNum].Username], `{"action":"player-call"}`)
		g = waitGame(t, aRecv.games, func(g poker.GameView) bool { return !g.Betting })
		for _, p := range g.Players {
			if p.Stack == 0 {
				busted = p.Username
			}
		}
		if busted != "" {
			break
		}
		g = waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting })
		first = g.ActionNum
		pusher, stack = g.Players[first].Username, g.Players[first].Stack
	}

	send(t, conns[busted], `{"action":"rebuy","amount":99}`)
	if msg := waitError(t, recvs[busted]); msg != "rebuy must be at least 100" {
		t.Errorf("busted rebuy below min: got %q", msg)
	}
	send(t, conns[busted], `{"action":"rebuy","amount":100}`)
	waitLog(t, aRecv, busted+" rebuys for 100")
	waitGame(t, aRecv.games, func(g poker.GameView) bool {
		in, _ := byName(t, g, busted)
		return g.Betting && in
	})
}

// customSettings is a heads-up table with no turn timer.
const customSettings = `{"seats":2,"sb":25,"bb":50,"minBuy":1000,"maxBuy":2000,"turnTime":0,"nextHandDelay":2000}`

func createWithSettings(table, name, settings string) string {
	return `{"action":"create-table","tablename":"` + table + `","username":"` + name + `","settings":` + settings + `}`
}

func TestCreateWithSettings(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	send(t, a, createWithSettings("t", "a", customSettings))
	g := waitGame(t, aRecv.games, func(poker.GameView) bool { return true })
	want := tableSettings{Seats: 2, SmallBlind: 25, BigBlind: 50, MinBuy: 1000, MaxBuy: 2000, NextHandDelay: 2000}
	if got := <-aRecv.sets; got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
	if g.Config.BigBlind != 50 || g.Config.SmallBlind != 25 || g.Config.MaxBuy != 2000 {
		t.Errorf("game config = %+v, want blinds 25/50 and max buy 2000", g.Config)
	}

	send(t, a, `{"action":"take-seat","seatID":3,"buyIn":1000}`)
	if msg := waitError(t, aRecv); msg != "invalid seat" {
		t.Errorf("seat past the last: got %q", msg)
	}
	send(t, a, `{"action":"take-seat","seatID":1,"buyIn":999}`)
	if msg := waitError(t, aRecv); msg != "buy-in must be between 1000 and 2000" {
		t.Errorf("buy-in below min: got %q", msg)
	}
	send(t, a, `{"action":"take-seat","seatID":2,"buyIn":1000}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
}

func TestDefaultSettings(t *testing.T) {
	srv := newTestServer(t)
	_, aRecv := createAndWait(t, srv, "t", "a")
	if got := <-aRecv.sets; got != defaultSettings {
		t.Errorf("settings = %+v, want the defaults %+v", got, defaultSettings)
	}
}

func TestInvalidSettingsRejected(t *testing.T) {
	srv := newTestServer(t)
	tests := []struct {
		settings string
		want     string
	}{
		{`{"seats":10}`, "seats must be between 2 and 9"},
		{`{"bb":5}`, "small blind must be between 1 and the big blind"},
		{`"lots"`, "invalid settings"},
	}
	for _, tt := range tests {
		conn, r := dialRecv(t, srv)
		send(t, conn, createWithSettings("t", "a", tt.settings))
		if got := waitError(t, r); got != tt.want {
			t.Errorf("%s: got error %q, want %q", tt.settings, got, tt.want)
		}
	}
	// none of them made the table
	conn, r := dialRecv(t, srv)
	send(t, conn, joinMsg("t", "b"))
	if got := waitError(t, r); got != errNoTable.Error() {
		t.Errorf("joining after rejected creates: got %q, want %q", got, errNoTable.Error())
	}
}

func TestNoTurnTimer(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	send(t, a, createWithSettings("t", "a", customSettings), `{"action":"take-seat","seatID":1,"buyIn":1000}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 1 })
	b, _ := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), `{"action":"take-seat","seatID":2,"buyIn":1000}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return len(g.Players) == 2 })
	send(t, a, `{"action":"start-game"}`)
	waitGame(t, aRecv.games, func(g poker.GameView) bool { return g.Betting })
	if ends := <-aRecv.ends; ends != 0 {
		t.Errorf("turnEndsIn = %d with no turn timer, want 0", ends)
	}

	for _, l := range collectLogs(aRecv, turnTimeout+300*time.Millisecond) {
		if strings.HasSuffix(l, "(out of time)") {
			t.Errorf("acted for a player with no turn timer: %q", l)
		}
	}
}

func createWithPassword(table, name, password string) string {
	return `{"action":"create-table","tablename":"` + table + `","username":"` + name + `","password":"` + password + `"}`
}

func joinWithPassword(table, name, password string) string {
	return `{"action":"join-table","tablename":"` + table + `","username":"` + name + `","password":"` + password + `"}`
}

func TestTablePassword(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	send(t, a, createWithPassword("t", "a", "secret"))
	waitGame(t, aRecv.games, func(poker.GameView) bool { return true })
	if !<-aRecv.locked {
		t.Error("hasPassword = false on a table with a password")
	}
	if pw := <-aRecv.joined; pw != "secret" {
		t.Errorf("host joined with password %q, want %q", pw, "secret")
	}

	tests := []struct {
		msg  string
		code string
	}{
		{joinMsg("t", "b"), "password-required"},
		{joinWithPassword("t", "b", "Secret"), "wrong-password"},
		{joinWithPassword("t", "a", "nope"), "wrong-password"}, // not name taken, which would leak names
	}
	for _, tt := range tests {
		conn, r := dialRecv(t, srv)
		send(t, conn, tt.msg)
		waitError(t, r)
		if code := <-r.codes; code != tt.code {
			t.Errorf("%s: got code %q, want %q", tt.msg, code, tt.code)
		}
	}

	b, bRecv := dialRecv(t, srv)
	send(t, b, joinWithPassword("t", "b", "secret"))
	waitGame(t, bRecv.games, func(poker.GameView) bool { return true })
	if pw := <-bRecv.joined; pw != "secret" {
		t.Errorf("b joined with password %q, want %q", pw, "secret")
	}
}

func TestNoPasswordByDefault(t *testing.T) {
	srv := newTestServer(t)
	_, aRecv := createAndWait(t, srv, "t", "a")
	if <-aRecv.locked {
		t.Error("hasPassword = true on a table without one")
	}
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinWithPassword("t", "b", "anything"))
	waitGame(t, bRecv.games, func(poker.GameView) bool { return true })
}

func TestInvalidPasswordRejected(t *testing.T) {
	srv := newTestServer(t)
	conn, r := dialRecv(t, srv)
	send(t, conn, createWithPassword("t", "a", strings.Repeat("a", maxPasswordLen+1)))
	if got := waitError(t, r); got != "invalid password" {
		t.Errorf("long password: got %q", got)
	}
}

func TestReconnectSkipsPassword(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := dialRecv(t, srv)
	send(t, a, createWithPassword("t", "a", "secret"))
	waitGame(t, aRecv.games, func(poker.GameView) bool { return true })
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinWithPassword("t", "b", "secret"), seatMsg(2))
	token := waitToken(t, bRecv)

	b.Close()
	b2, b2Recv := dialRecv(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"b","token":"`+token+`"}`)
	g := waitGame(t, b2Recv.games, func(poker.GameView) bool { return true })
	if len(g.Players) != 1 || g.Players[0].Username != "b" {
		t.Errorf("players = %+v, want b reseated", g.Players)
	}
	if pw := <-b2Recv.joined; pw != "secret" {
		t.Errorf("reconnected b got password %q, want %q", pw, "secret")
	}
}

// waitLedger returns the first ledger matching cond.
func waitLedger(t *testing.T, r received, cond func(ledgerView) bool) ledgerView {
	t.Helper()
	timeout := time.After(2 * time.Second)
	var last ledgerView
	for {
		select {
		case l := <-r.ledger:
			if cond(l) {
				return l
			}
			last = l
		case <-timeout:
			t.Fatalf("timed out waiting for ledger, last: %+v", last)
		}
	}
}

// row is name's ledger row, or a zero row if it has none.
func row(l ledgerView, name string) ledgerRow {
	for _, r := range l.Rows {
		if r.Name == name {
			return r
		}
	}
	return ledgerRow{}
}

func TestLedgerBuyInsAndCashOuts(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	b, _ := dialRecv(t, srv)
	send(t, a, seatMsg(1), `{"action":"rebuy","amount":50}`)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return len(l.Rows) == 2 && l.Rows[0].BoughtIn == 150 })
	if want := (ledgerRow{Name: "a", BoughtIn: 150, Stack: 150}); l.Rows[0] != want {
		t.Errorf("a = %+v, want %+v", l.Rows[0], want)
	}
	if l.Banker != "a" || l.Balance != 0 {
		t.Errorf("banker %q, balance %d, want a and 0", l.Banker, l.Balance)
	}

	// b cashes out on leaving the seat, and buys in again
	send(t, b, `{"action":"leave-seat"}`)
	l = waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").CashedOut == 100 })
	if want := (ledgerRow{Name: "b", BoughtIn: 100, CashedOut: 100}); row(l, "b") != want {
		t.Errorf("b after leaving = %+v, want %+v", row(l, "b"), want)
	}
	send(t, b, `{"action":"take-seat","seatID":2,"buyIn":200}`)
	l = waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 200 })
	if want := (ledgerRow{Name: "b", BoughtIn: 300, CashedOut: 100, Stack: 200}); row(l, "b") != want {
		t.Errorf("b after sitting again = %+v, want %+v", row(l, "b"), want)
	}
}

func TestLedgerDuringHand(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv, _, _ := seatTwo(t, srv)

	// the blinds are bet, but still count as each player's chips
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return len(l.Rows) == 2 })
	for _, r := range l.Rows {
		if r.Stack != 100 || r.Net != 0 {
			t.Errorf("%s mid-hand = %+v, want stack 100 and net 0", r.Name, r)
		}
	}

	// a folds the small blind to b
	send(t, a, `{"action":"player-fold"}`)
	l = waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "a").Net != 0 })
	if row(l, "a").Net != -10 || row(l, "b").Net != 10 || l.Balance != 0 {
		t.Errorf("after the fold: a %+v, b %+v, balance %d", row(l, "a"), row(l, "b"), l.Balance)
	}
}

func TestLedgerKeptByName(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	b := dial(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 100 })

	// b leaves the table, then someone joins as b and continues b's row
	send(t, b, `{"action":"leave-table","tablename":"t"}`)
	waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").CashedOut == 100 })
	b2 := dial(t, srv)
	send(t, b2, joinMsg("t", "b"), seatMsg(2))
	waitChat(t, aRecv, gameAdminName+": b rejoined, continuing their ledger")
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 100 })
	if want := (ledgerRow{Name: "b", BoughtIn: 200, CashedOut: 100, Stack: 100}); row(l, "b") != want || len(l.Rows) != 2 {
		t.Errorf("rows = %+v, want b continued as %+v", l.Rows, want)
	}
}

func TestLedgerCashOutAfterGrace(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	b := dial(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").Stack == 100 })

	// a dropped connection keeps the seat and chips until the grace period ends
	b.Close()
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "b").CashedOut == 100 })
	if l.Balance != 0 {
		t.Errorf("balance = %d after grace, want 0", l.Balance)
	}
}

// waitChat waits for the client to receive chat msg, as "name: message".
func waitChat(t *testing.T, r received, msg string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case m := <-r.chat:
			if m == msg {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for chat %q", msg)
		}
	}
}

// waitAway waits for a game update whose disconnected players are names.
func waitAway(t *testing.T, r received, names ...string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case a := <-r.away:
			if slices.Equal(a, names) || len(a) == 0 && len(names) == 0 {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for disconnected %q", names)
		}
	}
}

func TestDisconnectedPlayers(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, seatMsg(1))
	b, bRecv := dialRecv(t, srv)
	send(t, b, joinMsg("t", "b"), seatMsg(2))
	token := waitToken(t, bRecv)

	b.Close()
	waitAway(t, aRecv, "b")
	b2 := dial(t, srv)
	send(t, b2, `{"action":"join-table","tablename":"t","username":"b","token":"`+token+`"}`)
	waitAway(t, aRecv)
}
