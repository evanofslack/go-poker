# Audit log and admin page

**Status: planned.** Stage 20 in `todo.md`.

Disputes ("who was really behind bob?", "did someone try to act out of turn?") can't be answered
today. The table log already records every host and seat action publicly, but by name only: no
person id, IP or connection. Rejected commands leave no trace: `client.go` refusals and `Reduce`
errors just go back to the sender. So: a saved `audit` table, and a read-only admin page for it and
the live tables, on a private port.

## Decisions
- Admin is its own listener, `ADMIN_ADDR`, off by default, like `METRICS_ADDR`. No auth code: reach
  it with an SSH tunnel, Tailscale or the Docker network, never the public port.
- Read-only first. Admin actions come later.
- Raw IPs, rows pruned after 30 days (`AUDIT_RETAIN_DAYS`).
- Audit rows outlive their table: `Prune` doesn't touch them, so closed tables can still be looked up.
- Accepted `Play`, `Chat` and `GetHands` aren't recorded: hands already hold every action, and chat
  stays out. Every other command is, and every refused one of any kind.
- Same SQL on SQLite and Postgres (`sqlStore`). The memory store keeps the last 10,000 entries.
- The admin page polls. No websocket.

## PR 1: audit log (`feat/audit`)

**Entry** (`server/audit.go`):
```go
type AuditEntry struct {
	ID        int64           // assigned by the auditor
	Time      int64           // unix ms
	Table     string          // table id, empty before joining one
	TableName string
	Person    string          // person id, empty before joining
	Name      string
	IP        string
	Conn      string          // per-connection id
	Kind      string          // kebab-case: "take-seat", "host-action", "join", "limit"
	Error     string          // why it was refused, empty if accepted
	Data      json.RawMessage // the command, secrets removed, and the log lines it added
}
```

**Migration** `002_audit.sql`: `audit(id integer primary key, time bigint, table_id, table_name,
person, name, ip, conn, kind, error, data text)`, indexed on `(table_id, id)`, `ip`, `name` and
`time`. No foreign key to `tables`. Ids come from the app, since identity columns differ between
Postgres and SQLite.

**Store**: `AppendAudit(ctx, []AuditEntry)`, `Audit(ctx, AuditQuery)` (by table id or name, person,
name, ip, kind, refused only, before an id, a limit; newest first), `PruneAudit(ctx, before)`,
`LastAuditID(ctx)`. `Prune` stays as it is.

**Auditor** (`hub.go`): `hub.audit(e)` is a non-blocking send to a buffered channel, dropped and
counted (`auditDropped`) when full. One goroutine numbers entries from `LastAuditID`, writes batches
every second or 100 entries, keeps a failed batch for the next try (capped), and flushes in
`hub.shutdown`. `hub.prune` also runs `PruneAudit`. It's apart from checkpoints, so `save` doesn't
change and refusals mid-hand are kept.

**What's recorded**
- `table.Result` gets `Err *Error`, set where `Reduce` already sends the error.
- `tableActor.handle`, for commands from a client (not timers or bots): kind from the command's type,
  person and name, the client's ip and connection id, `res.Err` (or "panic"), and as data the command
  with `Join.Password`, `Join.Token` and `SetPassword.Password` removed, the log lines it added
  (`LogSeq` before and after), and `reclaimed` from `res.Joined`.
- `client.go`: `Client` gets an `id` and `since`. Refusals and bad messages (bad JSON, unknown action,
  not joined) are recorded with `in.Action` as the kind; refused joins with the table name.
- Limits: the 429 in `serveWs` and the rate limit close, kind `limit`.

**Tests**: `TestStores` appends, queries, prunes and numbers on every store, and `Prune` leaves audit
rows. A raise out of turn is recorded with its ip and error; a client-side refusal is recorded; a
reconnect with a token is recorded as reclaimed; passwords and tokens never reach `data`; accepted
`Play` and `Chat` aren't recorded.

## PR 2: admin API (`feat/admin-api`)
- Config: `ADMIN_ADDR` (empty for off), `ADMIN_ORIGINS` (CORS, for the dev server only).
- `mountStatic`'s handler becomes `staticHandler(dir)`. The public router 404s `/admin`,
  `/admin.html` and `/admin/*`. The admin listener serves `/admin/api/*` and the same static files,
  so the page is same-origin.
- `envelope` gets `inspect func()`, run in the actor, so admin reads state and `conns` together
  without new table commands.
- `server/admin.go`, JSON:
  - `GET /admin/api/overview`: uptime, store type, connections in total and by IP, and every table
    (`table.Summary`, id, creator IP, status, people and connections, created).
  - `GET /admin/api/tables/{name}`: settings, seats (name, person, bot, stack, sitting out), people
    (id, name, seat, connected, ip, connection, since), host and banker, ledger (`table.TotalsView`),
    last logs.
  - `GET /admin/api/audit?table=&person=&name=&ip=&kind=&errors=&before=&limit=`: `Store.Audit`, so
    closed tables too.
- Response types in `server/adminapi.go`, added to the server package in `tygo.yaml`; `just types`.
- Tests: the public port 404s admin paths; the admin endpoints return a table's people with IPs, and
  an audit query by IP.

## PR 3: admin page (`ui/admin`)
- `web/pages/admin.tsx`, components in `web/components/Admin/`:
  - Overview: a table of tables, polled every 5s; a row opens the table.
  - Table: people with IP and connection, seats, ledger, and its audit (by person, kind, refused only;
    older pages by `before`).
  - Lookup: audit by IP or name across all tables, each row linking to its table.
- API base `NEXT_PUBLIC_ADMIN_URL`, empty for same-origin. Mantine controls, Tailwind layout, tokens.
- `justfile`: `backend` sets `ADMIN_ADDR=localhost:2113 ADMIN_ORIGINS=http://localhost:3000`, `web`
  sets `NEXT_PUBLIC_ADMIN_URL=http://localhost:2113`, `e2e` sets `ADMIN_ADDR=localhost:8098`.
- `e2e/admin.mjs`: create a table, send a refused action, then find the table, the person's IP and
  the refused row on the admin page.
- README: the env vars and how to reach the port.

## Later: admin actions
Close a table, drop a connection, remove a player: admin commands through the actor's inbox, each
recorded with kind `admin-*`.

## Checks
`just check` for each PR (`TestStores` needs Docker for Postgres). By hand: play from two browsers,
reconnect one, raise out of turn from devtools, then look at `audit` with `sqlite3` (PR 1) and on
`localhost:3000/admin` (PR 3); `localhost:8080/admin` 404s. `just e2e admin`, and the whole suite
before PR 3.
