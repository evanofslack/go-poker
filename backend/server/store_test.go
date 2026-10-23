package server

import (
	"context"
	"io"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/table"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestStores runs the same checks on every store. Postgres runs in a
// container, and is skipped without Docker.
func TestStores(t *testing.T) {
	t.Run("memory", func(t *testing.T) {
		testStore(t, newMemStore())
	})
	t.Run("sqlite", func(t *testing.T) {
		testStore(t, openTestStore(t, "sqlite:"+filepath.Join(t.TempDir(), "poker.db")))
	})
	t.Run("postgres", func(t *testing.T) {
		testcontainers.SkipIfProviderIsNotHealthy(t)
		ctx := context.Background()
		ctr, err := postgres.Run(ctx, "postgres:17-alpine", postgres.BasicWaitStrategies())
		testcontainers.CleanupContainer(t, ctr)
		if err != nil {
			t.Fatal(err)
		}
		url, err := ctr.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
		testStore(t, openTestStore(t, url))
	})
}

func openTestStore(t *testing.T, url string) Store {
	t.Helper()
	s, err := openStore(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := s.(io.Closer); ok {
		t.Cleanup(func() { c.Close() })
	}
	return s
}

func testHand(n int) Hand {
	return Hand{
		Record: table.HandRecord{Hand: n, Started: int64(n), Dealer: "al", Players: []table.HandPlayer{{Name: "al", Seat: 1, Stack: 100}}},
		Stats:  table.HandStats{Hand: n, Started: int64(n), Stacks: map[string]uint{"al": 100}, Played: []string{"al"}, Won: map[string]uint{}},
	}
}

func testStore(t *testing.T, s Store) {
	ctx := context.Background()
	at := time.UnixMilli(1_000_000)
	open := func() map[string]Saved {
		t.Helper()
		saved, err := s.Open(ctx)
		if err != nil {
			t.Fatal(err)
		}
		byName := map[string]Saved{}
		for _, sv := range saved {
			byName[sv.Name] = sv
		}
		return byName
	}
	save := func(c Checkpoint) {
		t.Helper()
		c.At = at
		if err := s.Save(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	// a new table, then more hands and logs; a hand saved twice is kept once
	logs := []table.Log{{ID: 1, Message: "one", Time: 1}, {ID: 2, Message: "two", Time: 2}}
	save(Checkpoint{ID: "a", Name: "t", Version: 1, State: []byte(`{"n":1}`), Hands: []Hand{testHand(1), testHand(2)}, Logs: logs})
	save(Checkpoint{ID: "a", Name: "t", Version: 1, State: []byte(`{"n":2}`), Hands: []Hand{testHand(2), testHand(3)}})
	got := open()["t"]
	want := Saved{ID: "a", Name: "t", Version: 1, State: []byte(`{"n":2}`), Hands: []Hand{testHand(1), testHand(2), testHand(3)}, Logs: logs}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}

	// Open returns the newest hands and logs
	var hands []Hand
	for n := 4; n <= table.MaxHands+5; n++ {
		hands = append(hands, testHand(n))
	}
	logs = nil
	for id := 3; id <= table.MaxLogs+5; id++ {
		logs = append(logs, table.Log{ID: id, Message: "x", Time: int64(id)})
	}
	save(Checkpoint{ID: "a", Name: "t", Version: 1, State: []byte(`{}`), Hands: hands, Logs: logs})
	got = open()["t"]
	if len(got.Hands) != table.MaxHands || got.Hands[0].Record.Hand != 6 || got.Hands[len(got.Hands)-1].Record.Hand != table.MaxHands+5 {
		t.Fatalf("%d hands from %d", len(got.Hands), got.Hands[0].Record.Hand)
	}
	if len(got.Logs) != table.MaxLogs || got.Logs[0].ID != 6 {
		t.Fatalf("%d logs from %d", len(got.Logs), got.Logs[0].ID)
	}

	// a closed table isn't open, and its name can be used again
	if err := s.MarkClosed(ctx, "a", "idle", at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkClosed(ctx, "nope", "idle", at); err != nil {
		t.Fatal(err)
	}
	if len(open()) != 0 {
		t.Fatalf("open after closing: %v", open())
	}
	save(Checkpoint{ID: "b", Name: "t", Version: 1, State: []byte(`{}`), Hands: []Hand{testHand(1)}})
	if got := open()["t"]; got.ID != "b" || len(got.Hands) != 1 || len(got.Logs) != 0 {
		t.Fatalf("reused name got %+v", got)
	}

	// pruning deletes closed tables older than the retention
	if n, err := s.Prune(ctx, at.Add(time.Hour)); err != nil || n != 0 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if n, err := s.Prune(ctx, at.Add(2*time.Hour)); err != nil || n != 1 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if got := open()["t"]; got.ID != "b" || len(got.Hands) != 1 {
		t.Fatalf("pruning touched the open table: %+v", got)
	}
}
