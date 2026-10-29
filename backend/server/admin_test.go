package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/evanofslack/go-poker/table"
)

// testSite is a static site with a home page and the admin page.
func testSite(t *testing.T) string {
	dir := t.TempDir()
	for name, body := range map[string]string{"index.html": "home", "admin.html": "admin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(b)
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	code, body := get(t, url)
	if code != http.StatusOK {
		t.Fatalf("%s: %d %s", url, code, body)
	}
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatal(err)
	}
}

func TestPublicHidesAdmin(t *testing.T) {
	srv := httptest.NewServer(staticHandler(testSite(t), true))
	defer srv.Close()
	for _, p := range []string{"/admin", "/admin.html", "/admin/api/overview"} {
		if code, _ := get(t, srv.URL+p); code != http.StatusNotFound {
			t.Errorf("%s: %d", p, code)
		}
	}
	if _, body := get(t, srv.URL+"/"); body != "home" {
		t.Errorf("home page %q", body)
	}
}

func TestAdmin(t *testing.T) {
	ws, hub := newTestServerHub(t, newMemStore())
	srv := httptest.NewServer(adminRouter(hub, testSite(t)))
	defer srv.Close()

	host := create(t, ws, "t", "host")
	a := join(t, ws, "t", "a")
	a.send(seatMsg(1))
	host.update(func(v table.View) bool { return len(v.Players) == 1 })
	a.send(`{"action":"take-seat","seat":2,"amount":-1}`)
	a.errorMsg()

	if _, body := get(t, srv.URL+"/admin"); body != "admin" {
		t.Errorf("admin page %q", body)
	}

	var o AdminOverview
	getJSON(t, srv.URL+"/admin/api/overview", &o)
	if o.Store != "memory" || o.Connections != 2 || o.ConnsByIP["127.0.0.1"] != 2 || len(o.Tables) != 1 {
		t.Fatalf("overview %+v", o)
	}
	if row := o.Tables[0]; row.Summary.Name != "t" || row.Connections != 2 || row.CreatorIP != "127.0.0.1" || row.Summary.Players != 1 {
		t.Errorf("table row %+v", row)
	}

	var tb AdminTable
	getJSON(t, srv.URL+"/admin/api/tables/t", &tb)
	if tb.Host != "host" || len(tb.People) != 2 || len(tb.View.Players) != 1 || len(tb.Ledger) != 1 {
		t.Fatalf("table %+v", tb)
	}
	pa := tb.People[0]
	if pa.Name != "a" || pa.Seat != 1 || !pa.Connected || pa.IP != "127.0.0.1" || pa.Conn == "" || pa.Since == 0 {
		t.Errorf("person %+v", pa)
	}
	if code, _ := get(t, srv.URL+"/admin/api/tables/nope"); code != http.StatusNotFound {
		t.Errorf("missing table: %d", code)
	}

	eventually(t, func() bool { return len(audited(t, hub, AuditQuery{Refused: true})) == 1 }, "refused buy-in not recorded")
	var entries []AuditEntry
	getJSON(t, srv.URL+"/admin/api/audit?table=t&refused=true", &entries)
	if len(entries) != 1 || entries[0].Name != "" || entries[0].Person != pa.ID || entries[0].Conn != pa.Conn {
		t.Fatalf("refused %+v", entries)
	}
	getJSON(t, srv.URL+"/admin/api/audit?ip=127.0.0.1&limit=2", &entries)
	if len(entries) != 2 || entries[0].Kind != "take-seat" || entries[1].Kind != "take-seat" {
		t.Fatalf("by ip %+v", entries)
	}
	getJSON(t, srv.URL+"/admin/api/audit?name=nobody", &entries)
	if entries == nil || len(entries) != 0 {
		t.Fatalf("none %+v", entries)
	}
	if code, _ := get(t, srv.URL+"/admin/api/audit?before=x"); code != http.StatusBadRequest {
		t.Errorf("bad before: %d", code)
	}
}

func TestAdminClosed(t *testing.T) {
	ws, hub := newTestServerHub(t, newMemStore())
	srv := httptest.NewServer(adminRouter(hub, testSite(t)))
	defer srv.Close()

	host := create(t, ws, "t", "host")
	host.send(seatMsg(1))
	host.update(func(v table.View) bool { return len(v.Players) == 1 })
	host.send(`{"action":"leave-table"}`)

	var closed []ClosedTable
	eventually(t, func() bool {
		getJSON(t, srv.URL+"/admin/api/closed", &closed)
		return len(closed) == 1
	}, "the table isn't listed as closed")
	if c := closed[0]; c.Name != "t" || c.Reason != "empty" || c.Created == 0 || c.Closed < c.Created {
		t.Fatalf("closed %+v", c)
	}

	var c AdminClosed
	getJSON(t, srv.URL+"/admin/api/closed/"+closed[0].ID, &c)
	if c.Ledger == nil || len(c.Ledger.Rows) != 1 || c.Ledger.Rows[0].CashedOut != 1000 || len(c.Logs) == 0 {
		t.Fatalf("closed table %+v, ledger %+v", c, c.Ledger)
	}
	if code, _ := get(t, srv.URL+"/admin/api/closed/nope"); code != http.StatusNotFound {
		t.Errorf("missing table: %d", code)
	}
	if code, _ := get(t, srv.URL+"/admin/api/closed?before=x"); code != http.StatusBadRequest {
		t.Errorf("bad before: %d", code)
	}
}
