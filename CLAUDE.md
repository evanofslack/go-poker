# go-poker

Realtime multiplayer poker: Go websocket server (`backend/`) + Next.js client (`web/`).

## Commands
- Dev: `make start` (go on :8080, next on :3000). Needs `cd web && npm install` first.
- Backend (run from `backend/`): `gofmt -l .` (must be empty), `go vet ./...`, `go test -race ./...`
- Web (run from `web/`): `npm run type-check`, `npm run build`. No test suite.
- Run the checks for whichever side you touched before calling it done. CI runs exactly these.

## Architecture
- `backend/poker/`: game engine, forked from riverboat. Keep its license header.
- `backend/server/`: hub → tables → clients. Each table runs its own goroutine and game actions
  are serialized through it. `hub.tables` is guarded by `hub.mu`. Don't touch game state from
  client goroutines.
- Clients get per-player state via `GeneratePlayerView`, which hides other players' hole cards.
  Never broadcast the raw game state.
- `web/` is a static export (`output: 'export'`) served by the Go binary from `STATIC_DIR`.
  No API routes, SSR, or middleware.
- Tailwind 4: config lives in CSS, there is no tailwind.config.js.

## Websocket protocol
No shared schema. Changing a message means updating all three:
`backend/server/messages.go` ↔ `web/actions/actions.ts` ↔ `web/interfaces/index.ts`.
New inbound actions also need validation in `backend/server/validate.go` and a test.

## Conventions
- Commits: short, lowercase, imperative ("fix engine index panics").
