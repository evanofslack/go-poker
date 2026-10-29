# go-poker

Real time multiplayer poker application powered by go, react, and websockets

## demo

https://poker.evanslack.dev

## ui

<img width="950" alt="gopoker-screenshot" src="https://github.com/user-attachments/assets/dd306e9a-4b0e-4c6b-90e7-902216e41705" />


## getting started

Can run from a prebuilt container, `evanofslack/go-poker:latest`, with Postgres to keep tables
across restarts:

```yaml
services:
  go-poker:
    image: evanofslack/go-poker:latest
    restart: unless-stopped
    ports:
      - 8080:8080
    environment:
      DATABASE_URL: postgres://poker:poker@postgres:5432/poker?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: poker
      POSTGRES_PASSWORD: poker
      POSTGRES_DB: poker
    volumes:
      - postgres:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "poker"]
      interval: 2s
      retries: 15

volumes:
  postgres:
```

## storage

`DATABASE_URL` picks where tables are kept:

- unset: in memory, so a restart ends every game
- `postgres://user:pass@host:5432/db`: Postgres
- `sqlite:/data/poker.db`: a SQLite file; mount a volume at its directory

Tables are saved between hands and on shutdown. A graceful restart carries on mid-hand; a crash goes
back to the end of the last hand, which is then dealt again. Closed tables are kept for
`RETAIN_DAYS` (default 30) and then deleted.

The same store keeps an audit of what each connection did and tried, refused messages included, with
its IP: for settling disputes, never shown to players. Entries outlive their table and are deleted
after `AUDIT_RETAIN_DAYS` (default 30).

## admin

`ADMIN_ADDR` (off by default) serves a read-only admin page at `/admin` on its own address: open
tables, who is behind each name with their IP, and the audit. It has no login, so keep it off the
internet: bind it to `127.0.0.1:2113` and reach it over an SSH tunnel (`ssh -L 2113:localhost:2113
host`), or to a private network such as Tailscale. In Docker, don't publish its port. The public port
never serves it. With `just start` it's at `localhost:3000/admin`, and with `just serve` at
`localhost:2113/admin`.

## metrics

Prometheus metrics are served at `/metrics` on a separate port, `METRICS_ADDR` (default `:2112`, or set it empty to turn off).

| metric | labels | |
|---|---|---|
| `poker_connections` | | open websockets |
| `poker_tables` | `state`: active, ended | open tables |
| `poker_players` | `kind`: human, bot | seated players |
| `poker_limit` | `limit`: conns, conns_per_ip, tables, tables_per_ip, messages | configured limits |
| `poker_limit_hits_total` | `limit` | connections, tables or messages refused by a limit |
| `poker_hands_total` | | hands dealt |
| `poker_tables_closed_total` | `reason`: empty, ended, idle, shutdown | tables closed |
| `poker_seat_grace_total` | `outcome`: reconnected, expired | dropped players who came back or lost their seat |
| `poker_slow_client_drops_total` | | clients dropped for falling behind |
| `poker_panics_total` | | panics recovered while handling a message |
| `poker_save_errors_total` | | tables or audit batches that failed to save |
| `poker_audit_dropped_total` | | audit entries dropped for a full queue or a failing store |
| `poker_save_seconds` | | how long saving a table took |

Plus the standard `go_*` and `process_*` metrics.

## development

Run the go server and next dev server with hot reload (needs go, node and [just](https://github.com/casey/just)):

```bash
just start
```

Then open http://localhost:3000. Run `just check` for the same checks as CI, or `just` to list all recipes. Or build and run the production image from local code:

```bash
docker compose -f docker-compose-dev.yaml up --build
```
