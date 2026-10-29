package server

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/evanofslack/go-poker/table"
	"github.com/go-chi/chi/v5"
)

// adminLogs is how many of a table's log lines the admin API sends.
const adminLogs = 50

// maxClosedPage is the most closed tables one request lists, and the default.
const maxClosedPage = 50

func queryInt(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q isn't a number", s)
	}
	return n, nil
}

// adminRouter serves the admin API and, for the admin page, the static site.
// It's only ever on ADMIN_ADDR, which is never the public port: it has no
// auth of its own.
func adminRouter(h *Hub, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(adminCORS)
	r.Get("/admin/api/overview", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, h.adminOverview())
	})
	r.Get("/admin/api/tables/{name}", func(w http.ResponseWriter, r *http.Request) {
		t, err := h.adminTable(chi.URLParam(r, "name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, t)
	})
	r.Get("/admin/api/closed", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		before, err1 := queryInt(v.Get("before"))
		limit, err2 := queryInt(v.Get("limit"))
		if err := cmp.Or(err1, err2); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if limit <= 0 || limit > maxClosedPage {
			limit = maxClosedPage
		}
		tables, err := h.store.Closed(r.Context(), before, int(limit))
		if err != nil {
			slog.Default().Error("read closed tables", "error", err)
			http.Error(w, "reading closed tables failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, nonNil(tables))
	})
	r.Get("/admin/api/closed/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, err := h.adminClosed(r.Context(), chi.URLParam(r, "id"))
		if errors.Is(err, errNoTable) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err != nil {
			slog.Default().Error("read closed table", "error", err)
			http.Error(w, "reading the table failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, c)
	})
	r.Get("/admin/api/audit", func(w http.ResponseWriter, r *http.Request) {
		q, err := auditQuery(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		entries, err := h.store.Audit(r.Context(), q)
		if err != nil {
			slog.Default().Error("read audit", "error", err)
			http.Error(w, "reading the audit failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, nonNil(entries))
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusFound)
	})
	r.Handle("/*", staticHandler(staticDir, false))
	return r
}

// adminCORS lets the origins in ADMIN_ORIGINS, the web dev server, read the
// API.
func adminCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); adminOrigins[o] {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Warn("encode admin response", "error", err)
	}
}

// auditQuery reads an AuditQuery from r's query string.
func auditQuery(r *http.Request) (AuditQuery, error) {
	v := r.URL.Query()
	q := AuditQuery{
		Table:   v.Get("table"),
		Person:  v.Get("person"),
		Name:    v.Get("name"),
		IP:      v.Get("ip"),
		Kind:    v.Get("kind"),
		Refused: v.Get("refused") == "true",
	}
	var err error
	if s := v.Get("before"); s != "" {
		if q.Before, err = strconv.ParseInt(s, 10, 64); err != nil {
			return q, errors.New("before must be an id")
		}
	}
	if s := v.Get("limit"); s != "" {
		if q.Limit, err = strconv.Atoi(s); err != nil {
			return q, errors.New("limit must be a number")
		}
	}
	return q, nil
}

func (h *Hub) adminOverview() AdminOverview {
	o := AdminOverview{Started: h.started.UnixMilli(), Store: storeKind(h.store), Tables: []AdminTableRow{}}
	h.connMu.Lock()
	o.Connections, o.ConnsByIP = h.total, maps.Clone(h.conns)
	h.connMu.Unlock()
	for _, a := range h.allTables() {
		row := AdminTableRow{ID: a.id, Summary: *a.summary.Load(), CreatorIP: a.creatorIP, Public: a.public.Load(), Ended: a.ended.Load()}
		if a.inspect(func() { row.Connections = len(a.conns) }) != nil {
			continue // closed since
		}
		o.Tables = append(o.Tables, row)
	}
	slices.SortFunc(o.Tables, func(a, b AdminTableRow) int { return cmp.Compare(a.Summary.Name, b.Summary.Name) })
	return o
}

// adminTable is the table named name, read inside its actor.
func (h *Hub) adminTable(name string) (AdminTable, error) {
	a := h.table(name)
	if a == nil {
		return AdminTable{}, errNoTable
	}
	var t AdminTable
	err := a.inspect(func() {
		s := a.state
		t = AdminTable{
			ID: s.ID, Name: s.Name, CreatorIP: s.CreatorIP, Host: s.Host, Banker: s.Banker,
			Settings: s.Settings,
			View:     table.ViewFor(s, "", time.Now()),
			People:   []AdminPerson{},
			Ledger:   nonNil(s.Ledger),
			Logs:     nonNil(s.Logs[max(len(s.Logs)-adminLogs, 0):]),
		}
		for _, p := range s.People {
			ap := AdminPerson{ID: p.ID, Name: p.Name, Seat: p.Seat, Connected: p.Connected}
			if c := a.conns[p.ID]; c != nil {
				ap.IP, ap.Conn, ap.Since = c.ip, c.id, c.since.UnixMilli()
			}
			t.People = append(t.People, ap)
		}
		slices.SortFunc(t.People, func(a, b AdminPerson) int { return cmp.Compare(a.Name, b.Name) })
	})
	if err != nil {
		return AdminTable{}, errNoTable
	}
	return t, nil
}

// adminClosed is what the store kept of the table with id.
func (h *Hub) adminClosed(ctx context.Context, id string) (AdminClosed, error) {
	sv, err := h.store.Load(ctx, id)
	if err != nil {
		return AdminClosed{}, err
	}
	c := AdminClosed{Logs: nonNil(sv.Logs[max(len(sv.Logs)-adminLogs, 0):])}
	var s table.State
	if sv.Version == table.Version && json.Unmarshal(sv.State, &s) == nil {
		ledger := table.Ledger(&s)
		c.Settings, c.Ledger = s.Settings, &ledger
	}
	return c, nil
}

// storeKind names the store for the overview.
func storeKind(s Store) string {
	if sq, ok := s.(*sqlStore); ok {
		return sq.kind
	}
	return "memory"
}
