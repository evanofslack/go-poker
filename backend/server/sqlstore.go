package server

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evanofslack/go-poker/table"
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx"
	_ "modernc.org/sqlite"             // registers "sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// openStore opens the store url names: postgres://… or sqlite:path, or
// memory when it's empty.
func openStore(ctx context.Context, url string) (Store, error) {
	var db *sql.DB
	var err error
	switch {
	case url == "":
		return newMemStore(), nil
	case strings.HasPrefix(url, "postgres://"), strings.HasPrefix(url, "postgresql://"):
		db, err = sql.Open("pgx", url)
	case strings.HasPrefix(url, "sqlite:"):
		file := strings.TrimPrefix(strings.TrimPrefix(url, "sqlite:"), "//")
		db, err = sql.Open("sqlite", "file:"+file+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
		// SQLite takes one writer at a time
		db.SetMaxOpenConns(1)
	default:
		return nil, errors.New("DATABASE_URL must start with postgres:// or sqlite:")
	}
	if err != nil {
		return nil, err
	}
	s := &sqlStore{db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// sqlStore keeps tables in Postgres or SQLite. Both take the same SQL.
type sqlStore struct {
	db *sql.DB
}

// migrate applies the migrations not yet applied, in order. Each is named
// for its version, like 001_tables.sql.
func (s *sqlStore) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `create table if not exists migrations (version integer primary key)`); err != nil {
		return err
	}
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	for _, f := range files {
		version, err := strconv.Atoi(strings.SplitN(path.Base(f), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		var n int
		if err := s.db.QueryRowContext(ctx, `select count(*) from migrations where version = $1`, version).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		b, err := migrations.ReadFile(f)
		if err != nil {
			return err
		}
		err = s.tx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(b)); err != nil {
				return fmt.Errorf("%s: %w", f, err)
			}
			_, err := tx.ExecContext(ctx, `insert into migrations (version) values ($1)`, version)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// tx runs f in a transaction, committing if it returns nil.
func (s *sqlStore) tx(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := f(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *sqlStore) Save(ctx context.Context, c Checkpoint) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		at := c.At.UnixMilli()
		_, err := tx.ExecContext(ctx, `
			insert into tables (id, name, version, state, created_at, updated_at) values ($1, $2, $3, $4, $5, $5)
			on conflict (id) do update set version = excluded.version, state = excluded.state, updated_at = excluded.updated_at`,
			c.ID, c.Name, c.Version, string(c.State), at)
		if err != nil {
			return err
		}
		for _, h := range c.Hands {
			record, err := json.Marshal(h.Record)
			if err != nil {
				return err
			}
			stats, err := json.Marshal(h.Stats)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `
				insert into hands (table_id, hand, record, stats) values ($1, $2, $3, $4)
				on conflict (table_id, hand) do update set record = excluded.record, stats = excluded.stats`,
				c.ID, h.Record.Hand, string(record), string(stats))
			if err != nil {
				return err
			}
		}
		for _, l := range c.Logs {
			_, err := tx.ExecContext(ctx, `
				insert into logs (table_id, id, message, time) values ($1, $2, $3, $4)
				on conflict (table_id, id) do update set message = excluded.message, time = excluded.time`,
				c.ID, l.ID, l.Message, l.Time)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqlStore) MarkClosed(ctx context.Context, id, reason string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `update tables set closed_at = $1, close_reason = $2 where id = $3 and closed_at is null`,
		at.UnixMilli(), reason, id)
	return err
}

func (s *sqlStore) Open(ctx context.Context) ([]Saved, error) {
	rows, err := s.db.QueryContext(ctx, `select id, name, version, state from tables where closed_at is null`)
	if err != nil {
		return nil, err
	}
	var saved []Saved
	for rows.Next() {
		var t Saved
		var state string
		if err := rows.Scan(&t.ID, &t.Name, &t.Version, &state); err != nil {
			rows.Close()
			return nil, err
		}
		t.State = []byte(state)
		saved = append(saved, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// after the tables are read: SQLite has only the one connection
	for i := range saved {
		if saved[i].Hands, err = s.hands(ctx, saved[i].ID); err != nil {
			return nil, err
		}
		if saved[i].Logs, err = s.logs(ctx, saved[i].ID); err != nil {
			return nil, err
		}
	}
	return saved, nil
}

// hands is a table's last table.MaxHands hands, oldest first.
func (s *sqlStore) hands(ctx context.Context, id string) ([]Hand, error) {
	rows, err := s.db.QueryContext(ctx, `select record, stats from hands where table_id = $1 order by hand desc limit $2`, id, table.MaxHands)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hands []Hand
	for rows.Next() {
		var h Hand
		var record, stats string
		if err := rows.Scan(&record, &stats); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(record), &h.Record); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(stats), &h.Stats); err != nil {
			return nil, err
		}
		hands = append(hands, h)
	}
	slices.Reverse(hands)
	return hands, rows.Err()
}

// logs is a table's last table.MaxLogs log lines, oldest first.
func (s *sqlStore) logs(ctx context.Context, id string) ([]table.Log, error) {
	rows, err := s.db.QueryContext(ctx, `select id, message, time from logs where table_id = $1 order by id desc limit $2`, id, table.MaxLogs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []table.Log
	for rows.Next() {
		var l table.Log
		if err := rows.Scan(&l.ID, &l.Message, &l.Time); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	slices.Reverse(logs)
	return logs, rows.Err()
}

func (s *sqlStore) Prune(ctx context.Context, closedBefore time.Time) (int, error) {
	var n int64
	err := s.tx(ctx, func(tx *sql.Tx) error {
		before := closedBefore.UnixMilli()
		for _, q := range []string{
			`delete from hands where table_id in (select id from tables where closed_at < $1)`,
			`delete from logs where table_id in (select id from tables where closed_at < $1)`,
		} {
			if _, err := tx.ExecContext(ctx, q, before); err != nil {
				return err
			}
		}
		res, err := tx.ExecContext(ctx, `delete from tables where closed_at < $1`, before)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	return int(n), err
}

func (s *sqlStore) AppendAudit(ctx context.Context, entries []AuditEntry) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		for _, e := range entries {
			_, err := tx.ExecContext(ctx, `
				insert into audit (id, time, table_id, table_name, person, name, ip, conn, kind, error, data)
				values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) on conflict (id) do nothing`,
				e.ID, e.Time, e.Table, e.TableName, e.Person, e.Name, e.IP, e.Conn, e.Kind, e.Error, string(e.Data))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqlStore) Audit(ctx context.Context, q AuditQuery) ([]AuditEntry, error) {
	var where []string
	var args []any
	add := func(cond string, arg any) {
		args = append(args, arg)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.Table != "" {
		add("(table_id = ? or table_name = ?)", q.Table)
	}
	for _, f := range []struct{ col, v string }{{"person", q.Person}, {"name", q.Name}, {"ip", q.IP}, {"kind", q.Kind}} {
		if f.v != "" {
			add(f.col+" = ?", f.v)
		}
	}
	if q.Refused {
		where = append(where, "error <> ''")
	}
	if q.Before > 0 {
		add("id < ?", q.Before)
	}
	query := `select id, time, table_id, table_name, person, name, ip, conn, kind, error, data from audit`
	if len(where) > 0 {
		query += " where " + strings.Join(where, " and ")
	}
	args = append(args, q.limit())
	query += " order by id desc limit $" + strconv.Itoa(len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var data string
		if err := rows.Scan(&e.ID, &e.Time, &e.Table, &e.TableName, &e.Person, &e.Name, &e.IP, &e.Conn, &e.Kind, &e.Error, &data); err != nil {
			return nil, err
		}
		if data != "" {
			e.Data = json.RawMessage(data)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *sqlStore) LastAuditID(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `select coalesce(max(id), 0) from audit`).Scan(&id)
	return id, err
}

func (s *sqlStore) PruneAudit(ctx context.Context, before time.Time) (int, error) {
	res, err := s.db.ExecContext(ctx, `delete from audit where time < $1`, before.UnixMilli())
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// Close closes the database.
func (s *sqlStore) Close() error {
	return s.db.Close()
}
