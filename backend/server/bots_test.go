package server

import (
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/poker"
)

func addBotMsg(seat int, name string) string {
	return addLevelBotMsg(seat, name, "easy")
}

func addLevelBotMsg(seat int, name, level string) string {
	return `{"action":"add-bot","seatID":` + strconv.Itoa(seat) + `,"buyIn":100,"name":"` + name + `","level":"` + level + `"}`
}

func removeBotMsg(seat int) string {
	return `{"action":"remove-bot","seatID":` + strconv.Itoa(seat) + `}`
}

// names are the usernames of the seated players in g.
func names(g poker.GameView) []string {
	var ns []string
	for _, p := range g.Players {
		ns = append(ns, p.Username)
	}
	return ns
}

func TestOnlyHostAddsBots(t *testing.T) {
	srv := newTestServer(t)
	host, hRecv := createAndWait(t, srv, "t", "host")
	guest, gRecv := dialRecv(t, srv)
	send(t, guest, joinMsg("t", "guest"))
	waitGame(t, gRecv.games, func(poker.GameView) bool { return true })

	send(t, guest, addBotMsg(1, "harper"))
	if got := waitError(t, gRecv); got != "only the host can add bots" {
		t.Fatalf("guest add-bot error = %q", got)
	}
	send(t, host, addBotMsg(1, "harper"))
	waitGame(t, hRecv.games, func(g poker.GameView) bool { return slices.Equal(names(g), []string{"harper"}) })

	send(t, guest, removeBotMsg(1))
	if got := waitError(t, gRecv); got != "only the host can remove bots" {
		t.Fatalf("guest remove-bot error = %q", got)
	}
	send(t, host, removeBotMsg(1))
	waitGame(t, hRecv.games, func(g poker.GameView) bool { return len(g.Players) == 0 })
}

func TestAddBotErrors(t *testing.T) {
	srv := newTestServer(t)
	host, r := createAndWait(t, srv, "t", "host")
	send(t, host, seatMsg(1))

	tests := []struct {
		msg  string
		want string
	}{
		{addBotMsg(1, "eric"), "seat taken"},
		{addBotMsg(0, "eric"), "invalid seat"},
		{addBotMsg(2, "host"), "name taken"},
		{addBotMsg(2, "system"), "invalid bot name"},
		{addBotMsg(2, " "), "invalid bot name"},
		{`{"action":"add-bot","seatID":2,"buyIn":5,"name":"eric","level":"easy"}`, "buy-in must be between 100 and 10000"},
		{`{"action":"add-bot","seatID":2,"buyIn":100,"name":"eric","level":"hard"}`, "invalid bot level"},
		{removeBotMsg(1), "no bot in that seat"}, // a person, not a bot
	}
	for _, tt := range tests {
		send(t, host, tt.msg)
		if got := waitError(t, r); got != tt.want {
			t.Errorf("%s: error = %q, want %q", tt.msg, got, tt.want)
		}
	}
}

// TestBotsPlayAlone checks the host can start a game of only bots without a
// seat, and the bots play hands through on their own.
func TestBotsPlayAlone(t *testing.T) {
	srv, hub := newTestServerHub(t)
	host, r := createAndWait(t, srv, "t", "host")
	send(t, host, addBotMsg(1, "harper"), addBotMsg(2, "eric"), `{"action":"start-game"}`)
	eventually(t, func() bool { return isRunning(t, hub, "t") }, "host could not start a bots-only game")

	// several hands, so both bots have acted and the server dealt again
	hands := 0
	timeout := time.After(5 * time.Second)
	for hands < 3 {
		select {
		case h := <-r.history:
			if !h.Full {
				hands++
			}
		case <-timeout:
			t.Fatalf("bots finished %d hands, want 3", hands)
		}
	}
}

func TestMediumBotsPlay(t *testing.T) {
	srv, hub := newTestServerHub(t)
	host, r := createAndWait(t, srv, "t", "host")
	send(t, host, addLevelBotMsg(1, "harper", "medium"), addLevelBotMsg(2, "eric", "medium"), addBotMsg(3, "gus"), `{"action":"start-game"}`)
	eventually(t, func() bool { return isRunning(t, hub, "t") }, "game didn't start")

	hands := 0
	timeout := time.After(5 * time.Second)
	for hands < 3 {
		select {
		case h := <-r.history:
			if !h.Full {
				hands++
			}
		case <-timeout:
			t.Fatalf("bots finished %d hands, want 3", hands)
		}
	}
}

func TestBotLevelsSent(t *testing.T) {
	srv := newTestServer(t)
	host, _ := createAndWait(t, srv, "t", "host")
	conn, r := dialRecv(t, srv)
	send(t, conn, joinMsg("t", "guest"))
	send(t, host, addLevelBotMsg(1, "harper", "medium"), addBotMsg(2, "gus"))

	timeout := time.After(2 * time.Second)
	for {
		select {
		case levels := <-r.bots:
			got := slices.Sorted(maps.Values(levels))
			if slices.Equal(got, []string{"easy", "medium"}) {
				return
			}
		case <-timeout:
			t.Fatal("bot levels not sent")
		}
	}
}

func TestBotOnlyTableDeleted(t *testing.T) {
	srv, hub := newTestServerHub(t)
	host, _ := createAndWait(t, srv, "t", "host")
	send(t, host, addBotMsg(1, "harper"), addBotMsg(2, "eric"), `{"action":"start-game"}`)
	eventually(t, func() bool { return isRunning(t, hub, "t") }, "game didn't start")
	host.Close()
	waitForTables(t, hub)
}

func TestBotPlaysPerson(t *testing.T) {
	srv := newTestServer(t)
	host, r := createAndWait(t, srv, "t", "host")
	send(t, host, seatMsg(1), addBotMsg(2, "harper"), `{"action":"start-game"}`)

	// the host checks or calls whenever it's their turn, and the bot plays back
	timeout := time.After(5 * time.Second)
	for {
		select {
		case g := <-r.games:
			if g.Betting && g.Players[g.ActionNum].Username == "host" {
				send(t, host, `{"action":"player-call"}`)
			}
			if g.Stage == poker.PreDeal && g.Running && slices.ContainsFunc(g.Pots, func(p poker.Pot) bool { return len(p.WinningPlayerNums) > 0 }) {
				return
			}
		case <-timeout:
			t.Fatal("hand against a bot didn't finish")
		}
	}
}

// TestBustedBotRebuys plays all-in hands between two bots until one busts,
// then checks it bought back in.
func TestBustedBotRebuys(t *testing.T) {
	tbl := newTable("t", defaultSettings)
	tbl.botDelay = func() time.Duration { return time.Hour } // this test moves for the bots
	go tbl.run()
	t.Cleanup(func() { close(tbl.done) })
	tbl.mu.Lock()
	for i, name := range []string{"harper", "eric"} {
		uuid := seatPlayer(tbl, name, uint(i+1), 100)
		tbl.bots[uuid] = botSeat{level: "easy", buyIn: 100}
	}
	if err := tbl.game.Start(); err != nil {
		t.Fatal(err)
	}
	tbl.mu.Unlock()

	eventually(t, func() bool {
		tbl.mu.Lock()
		defer tbl.mu.Unlock()
		view := tbl.game.GenerateOmniView()
		for _, p := range view.Players {
			if p.TotalBuyIn > 100 {
				return p.Stack == 100 && p.Ready
			}
		}
		if view.Betting {
			pn := view.ActionNum
			if raise(tbl, pn, view.Players[pn].Stack) != nil {
				call(tbl, pn)
			}
		}
		return false
	}, "busted bot didn't rebuy")
}

func TestBotBuyInInLedger(t *testing.T) {
	srv := newTestServer(t)
	a, aRecv := createAndWait(t, srv, "t", "a")
	send(t, a, addBotMsg(2, "harper"))
	l := waitLedger(t, aRecv, func(l ledgerView) bool { return row(l, "harper").BoughtIn == 100 })
	if r := row(l, "harper"); r.Stack != 100 || r.Net != 0 || l.Balance != 0 {
		t.Errorf("harper = %+v, balance %d; want 100 in and on the table", r, l.Balance)
	}
}
