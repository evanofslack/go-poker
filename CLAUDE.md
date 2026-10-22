# go-poker

Realtime multiplayer poker: Go websocket server (`backend/`) + Next.js client (`web/`).

## Commands
- Dev: `just start` (go on :8080, next on :3000; installs web deps if missing). `just serve` runs the
  production-like build on :8080. `just` lists all recipes.
- `just check` runs exactly what CI runs:
  - Backend (run from `backend/`): `gofmt -l .` (must be empty), `go vet ./...`, `go test -race ./...`
  - Web (run from `web/`): `npm run type-check`, `npm run build`. No test suite.
- Run the checks for whichever side you touched before calling it done.
- `just e2e [scripts...]` runs the browser tests (not in CI). Run the scripts for the area you
  touched while working, and all of them once before a PR that changes the web app. For visual
  changes, look at screenshots too (`SHOTS=<dir>`). See `e2e/README.md`.

## Architecture
- `backend/poker/`: game engine, forked from riverboat. Keep its license header.
- `backend/server/`: hub → tables → clients. Each table runs its own goroutine and game actions
  are serialized by `table.mu` (handlers run on client goroutines and must hold it). The table
  goroutine owns `table.clients` and fans out messages. `hub.tables` is guarded by `hub.mu`.
- Clients get per-player state via `GeneratePlayerView`, which hides other players' hole cards.
  Never broadcast the raw game state.
- `web/` is a static export (`output: 'export'`) served by the Go binary from `STATIC_DIR`.
  No API routes, SSR, or middleware.
- Tailwind 4: config lives in CSS, there is no tailwind.config.js.

## Websocket protocol
No shared schema. Changing a message means updating all three:
`backend/server/messages.go` ↔ `web/actions/actions.ts` ↔ `web/interfaces/index.ts`.
New inbound actions also need validation in `backend/server/validate.go` and a test.

## Metrics
Prometheus metrics live in `backend/server/metrics.go` and are served on `METRICS_ADDR` (default `:2112`),
never the public port. Gauges are read from the hub at scrape time; counters are incremented where things
happen. Labels must be fixed sets created at 0 in `init` (never a table, address or player). A new metric
also goes in the README's metrics table.

## Conventions
- Commits: short, lowercase, imperative ("fix engine index panics").
