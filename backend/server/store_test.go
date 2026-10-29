package server

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"slices"
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

	// closed tables are listed, last closed first, and load like open ones
	if err := s.MarkClosed(ctx, "b", "ended", at.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	closed, err := s.Closed(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	wantClosed := []ClosedTable{
		{ID: "b", Name: "t", Created: at.UnixMilli(), Closed: at.Add(3 * time.Hour).UnixMilli(), Reason: "ended", Hands: 1},
		{ID: "a", Name: "t", Created: at.UnixMilli(), Closed: at.Add(time.Hour).UnixMilli(), Reason: "idle", Hands: table.MaxHands + 5},
	}
	if !reflect.DeepEqual(closed, wantClosed) {
		t.Fatalf("closed %+v\nwant %+v", closed, wantClosed)
	}
	if closed, err := s.Closed(ctx, at.Add(3*time.Hour).UnixMilli(), 10); err != nil || len(closed) != 1 || closed[0].ID != "a" {
		t.Fatalf("closed before b: %+v, %v", closed, err)
	}
	if closed, err := s.Closed(ctx, 0, 1); err != nil || len(closed) != 1 || closed[0].ID != "b" {
		t.Fatalf("one closed: %+v, %v", closed, err)
	}
	if sv, err := s.Load(ctx, "a"); err != nil || sv.Name != "t" || len(sv.Hands) != table.MaxHands || len(sv.Logs) != table.MaxLogs {
		t.Fatalf("load a: %d hands, %d logs, %v", len(sv.Hands), len(sv.Logs), err)
	}
	if _, err := s.Load(ctx, "nope"); !errors.Is(err, errNoTable) {
		t.Fatalf("load nope: %v", err)
	}

	// pruning deletes closed tables older than the retention
	if n, err := s.Prune(ctx, at.Add(time.Hour)); err != nil || n != 0 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if n, err := s.Prune(ctx, at.Add(2*time.Hour)); err != nil || n != 1 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if closed, err := s.Closed(ctx, 0, 10); err != nil || len(closed) != 1 || closed[0].ID != "b" {
		t.Fatalf("pruning left %+v, %v", closed, err)
	}

	testAuditStore(t, s)
}

func testAuditStore(t *testing.T, s Store) {
	ctx := context.Background()
	ids := func(q AuditQuery) []int64 {
		t.Helper()
		got, err := s.Audit(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		var ids []int64
		for _, e := range got {
			ids = append(ids, e.ID)
		}
		return ids
	}
	if id, err := s.LastAuditID(ctx); err != nil || id != 0 {
		t.Fatalf("last id %d, %v", id, err)
	}
	entries := []AuditEntry{
		{ID: 1, Time: 1000, Table: "a", TableName: "t", Person: "p1", Name: "al", IP: "1.1.1.1", Conn: "c1", Kind: "join", Data: []byte(`{"token":false}`)},
		{ID: 2, Time: 2000, Table: "a", TableName: "t", Person: "p2", Name: "bo", IP: "2.2.2.2", Conn: "c2", Kind: "play-check", Error: "not your turn"},
		{ID: 3, Time: 3000, TableName: "u", IP: "2.2.2.2", Conn: "c3", Kind: "join-table", Error: "table not found"},
	}
	if err := s.AppendAudit(ctx, entries[:2]); err != nil {
		t.Fatal(err)
	}
	// a batch tried again after a failure writes each entry once
	if err := s.AppendAudit(ctx, entries); err != nil {
		t.Fatal(err)
	}
	if id, err := s.LastAuditID(ctx); err != nil || id != 3 {
		t.Fatalf("last id %d, %v", id, err)
	}
	got, err := s.Audit(ctx, AuditQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []AuditEntry{entries[2], entries[1], entries[0]}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	for _, c := range []struct {
		q    AuditQuery
		want []int64
	}{
		{AuditQuery{Table: "a"}, []int64{2, 1}},
		{AuditQuery{Table: "u"}, []int64{3}},
		{AuditQuery{IP: "2.2.2.2"}, []int64{3, 2}},
		{AuditQuery{Name: "al"}, []int64{1}},
		{AuditQuery{Person: "p2"}, []int64{2}},
		{AuditQuery{Kind: "join"}, []int64{1}},
		{AuditQuery{Refused: true}, []int64{3, 2}},
		{AuditQuery{Before: 3, Limit: 1}, []int64{2}},
	} {
		if got := ids(c.q); !slices.Equal(got, c.want) {
			t.Errorf("%+v: got %v, want %v", c.q, got, c.want)
		}
	}

	// the tables are pruned above, and their entries stay; age prunes them
	if n, err := s.PruneAudit(ctx, time.UnixMilli(2500)); err != nil || n != 2 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if got := ids(AuditQuery{}); !slices.Equal(got, []int64{3}) {
		t.Fatalf("after pruning %v", got)
	}
}
