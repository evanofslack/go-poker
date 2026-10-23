package server

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evanofslack/go-poker/table"
)

func TestMetricsHandler(t *testing.T) {
	hub := newHub(newMemStore())
	if _, err := hub.createTable("a", "1.1.1.1", "", table.DefaultSettings); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(metricsHandler(hub))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{
		`poker_tables{state="active"} 1`,
		`poker_tables{state="ended"} 0`,
		`poker_players{kind="human"} 0`,
		`poker_limit{limit="tables"}`,
		`poker_limit_hits_total{limit="messages"}`,
		`poker_tables_closed_total{reason="shutdown"}`,
		`poker_seat_grace_total{outcome="expired"}`,
		"poker_connections ",
		"poker_hands_total ",
		"poker_panics_total ",
		"poker_slow_client_drops_total ",
		"go_goroutines ",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("metrics missing %q", want)
		}
	}
}
