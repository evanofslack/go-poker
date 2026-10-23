# Persistence

Tables survive a server restart. Builds on the engine/table refactor (`refactor.md`): `table.State`
is plain JSON, each table actor already calls `Store.Save`, and `table.Restore` already readies a
loaded table (seated people get the reconnect grace, watchers go, the player to act gets a fresh
turn). Clients already rejoin with their seat token after a dropped connection.

## Decisions
- Postgres and SQLite behind one `Store`, picked by `DATABASE_URL`; memory stays the default so
  `just start` and the tests need nothing.
- A table is a small hot state blob plus records appended as they happen (hands, logs), not one
  growing blob.
- Saves happen at checkpoints: whenever no hand is in progress, and on shutdown. A crash mid-hand
  rolls the table back to before that hand was dealt.
- Closed tables are kept, marked closed, and pruned after `RETAIN_DAYS` (default 30). No UI reads
  them yet.
- One server process. No locking or leases between instances.

## Checkpoints
- `table.Result` gets `Checkpoint bool`: set when the command leaves the game between hands
  (`!Game.InHand()`). That covers hand ends, seating, rebuys, host actions and session end
  between hands. Commands mid-hand don't save.
- Rolling back is clean: the state saved after a hand already holds the deal timer, and the
  shuffle comes from the seed and hand number, so on restore the same hand is dealt again.
- Lost on a crash mid-hand: that hand, plus anything else done during it (someone sitting down,
  sit-out-next, a queued rebuy request). People rejoin and redo it.
- Shutdown saves the current state even mid-hand, so a deploy carries the hand on.
- Saves run in the actor, with a short timeout. A failed save is logged and counted in a metric;
  the table keeps playing and the next checkpoint writes everything not yet saved.

## Storage shape
- `State` gets an `ID` (random). Names can be reused once a table closes, so the ID is the key.
- Hot state: `State` without `History`, `Stats` and `Logs`. Small, rewritten at each checkpoint.
  `Totals` stay in it; they're per player and cover every hand.
- Hands (record and stats together) and log lines are appended. The actor tracks the last hand
  and log ID saved and sends only newer ones, so a checkpoint after a missed one catches up.
- Loading reassembles `State` with the newest 1000 hands and 200 log lines, as kept in memory
  today. Older ones stay in the database for the archive.

```sql
create table tables (
  id          text primary key,
  name        text not null,
  version     integer not null,      -- table.Version of state
  state       text not null,         -- hot state, JSON
  created_at  bigint not null,       -- unix ms
  updated_at  bigint not null,
  closed_at   bigint,                -- null while open
  close_reason text                  -- empty, idle, ended, incompatible
);
create unique index tables_open_name on tables (name) where closed_at is null;
create table hands (table_id text, hand integer, record text, stats text, primary key (table_id, hand));
create table logs  (table_id text, id integer, message text, time bigint, primary key (table_id, id));
```

The same schema works on both: JSON as text and times as unix ms. Partial indexes and
`on conflict` upserts exist in both.

## Store
```go
type Checkpoint struct {
    ID, Name string
    Version  int
    State    []byte             // hot state
    Hands    []table.HandDone   // new since the last save
    Logs     []table.Log        // new since the last save
}

type Store interface {
    Save(ctx context.Context, c Checkpoint) error // upsert the table, append hands and logs, one transaction
    MarkClosed(ctx context.Context, id, reason string, at time.Time) error
    Open(ctx context.Context) ([]Saved, error) // open tables, with their recent hands and logs
    Prune(ctx context.Context, closedBefore time.Time) (int, error)
}
```
- One `sqlStore` on `database/sql`, with the driver and the few dialect differences (placeholders)
  chosen by URL: `postgres://…` uses pgx's stdlib driver, and `sqlite:///path` uses
  `modernc.org/sqlite`, which needs no C compiler, so the Docker build stays the same. SQLite runs
  in WAL mode with one writer connection.
- `memStore` implements the same interface, keeping closed tables until pruned like the others.
- Schema migrations are numbered SQL files embedded in the binary, applied at startup and
  tracked in a `migrations` table.
- On restart, a saved table whose `version` doesn't match is closed with reason `incompatible`
  rather than loaded. Version upgrades can come later if a change is worth carrying tables
  across.

## Server
- Startup: open the store, run migrations, load open tables, `table.Restore` them, start their
  actors, then listen.
- When a table closes it's marked closed in the store, not deleted.
- A prune loop runs hourly.
- Config: `DATABASE_URL`, `RETAIN_DAYS`.
- Metrics: save errors and save duration.
- Docker: `docker-compose.yaml` gains a Postgres service with a volume, and sets `DATABASE_URL`.
  SQLite needs only a volume and a path, documented as the alternative.
- Shutdown tells clients `restarting` rather than `closed`, so they keep their seat token and
  rejoin once the server is back.
- Web: the "games aren't saved" line on the join page goes.

## Tests
- Store conformance suite, run against memory, SQLite (a temp file) and Postgres in a
  testcontainers container, in `go test` locally and in CI. Without Docker the Postgres run skips.
- Restart test: play to a checkpoint and part way through the next hand, then start a new hub on
  the same store without shutting down. The table is back at the checkpoint, redeals the same
  hand, and a token reclaims the seat. A second test shuts down gracefully mid-hand and gets the
  hand back as it was.
- e2e `restart.mjs`: run the server on SQLite; restart it gracefully mid-hand and check the hand
  carries on, then kill it after a hand and check seats and stacks are back at that hand's end.

## PRs
1. **checkpoints** — `Result.Checkpoint`, `State.ID`, the hot/records split, the new `Store`
   interface on `memStore`, close-instead-of-delete, prune loop, restart tests on memory.
2. **sql store** — `sqlStore` with migrations, SQLite and Postgres, conformance suite on
   testcontainers, compose, config, docs, e2e restart script.

The second needs only the interface from the first, so it can be written alongside it.
