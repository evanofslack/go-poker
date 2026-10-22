package server

import (
	"testing"
	"time"
)

func TestIdleTablesClosed(t *testing.T) {
	srv, hub := newTestServerHub(t)
	a, aRecv := createAndWait(t, srv, "a", "a")
	b, bRecv := createAndWait(t, srv, "b", "b")
	// bots playing don't count as activity
	send(t, a, addBotMsg(1, "bot1"), addBotMsg(2, "bot2"), `{"action":"start-game"}`)

	const idle = 300 * time.Millisecond
	time.Sleep(2 * idle)
	send(t, b, `{"action":"send-message","message":"still here"}`)
	waitChat(t, bRecv, "b: still here")
	hub.closeIdle(idle)

	if msg := waitError(t, aRecv); msg != "the table closed after 300ms idle" {
		t.Errorf("a got %q", msg)
	}
	if code := <-aRecv.codes; code != codeClosed {
		t.Errorf("a got code %q, want %q", code, codeClosed)
	}
	waitForTables(t, hub, "b")
}

func TestHumanDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		30 * time.Minute: "30 minutes",
		time.Minute:      "1 minute",
		2 * time.Hour:    "2 hours",
		90 * time.Minute: "90 minutes",
		time.Second:      "1s",
	} {
		if got := humanDuration(d); got != want {
			t.Errorf("humanDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
