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
- `backend/holdem/`: the game engine. Plain data `State`, no locks, clock or global rand; methods
  check legality before changing anything and return the `Event`s they caused. Hand evaluation
  comes from riverboat's `eval` module.
- `backend/table/`: one table as a reducer, `Reduce(state, now, cmd) Result`. People, seats,
  host controls, ledger, history, stats, bots. Timers are deadlines in `State.Timers`; the runtime
  fires them back as `TimerFired`. `State` is plain JSON, so a table can be saved and restored;
  History, Stats and Logs are left out of it and saved as they're appended.
- `backend/server/`: websockets and the runtime. Each table is an actor goroutine
  (`tableActor`) that owns its state and connections: inbox → `Reduce` → arm timers → send →
  save to the `Store` at checkpoints (between hands, and on shutdown). Client goroutines only
  talk to it through the inbox. See `plans/persistence.md`.
- Stores: memory by default; `DATABASE_URL` picks Postgres or SQLite (`sqlStore`, same SQL for
  both). Schema changes are new files in `backend/server/migrations/`, never edits to old ones.
  `TestStores` runs every store, Postgres via testcontainers (skipped without Docker).
- Clients get `table.ViewFor` and `table.EventsFor`, which hide other players' hole cards.
  Never send the raw state.
- `web/` is a static export (`output: 'export'`) served by the Go binary from `STATIC_DIR`.
  No API routes, SSR, or middleware.
- Tailwind 4: config lives in CSS, there is no tailwind.config.js.

## UI
- Mantine 8 for controls (`Button`, `ActionIcon`, …); Tailwind for layout, the table and the
  `phone:`/`tall:`/`short:` variants. Utility classes beat Mantine's styles (CSS layers), so
  `className` can adjust a Mantine component.
- One palette: the Mantine theme (`web/lib/theme.ts`), with the role tokens in
  `styles/index.css` pointing at its variables. The table and cards have their own tokens
  there (`felt`, `card`, `heart`, `gold`, …). Use tokens, not raw Tailwind colors.
- Dark by default, light by a toggle: the tokens' light values are in `styles/index.css` under
  `[data-mantine-color-scheme="light"]`. Prefer a light value for a token over the `dark:`
  variant, which is for the odd class that has to differ.
- Buttons: `filled` is the one main action in a group (join, start, save, approve), at most one
  per group. `default` (the theme's default) is everything else, the betting buttons included.
  `subtle` backs out (cancel, close). `outline` red opens something destructive, `filled` red
  confirms it. Icon-only is an `ActionIcon`.
- Sizes: `sm` (default); `xs` or `compact-*` in rows and headers; `md` with `bar` in the footer.
- Placement: an action that finishes a section sits left under what it acts on; a small action
  in a header or row sits at its right end; dialog footers are right-aligned.
- e2e: select buttons with `btn(text)` from `e2e/lib.mjs`, since Mantine nests the label.

## Websocket protocol
- `backend/server/messages.go` defines the messages; the types they carry live in `table` and
  `holdem`. `just types` (tygo) generates `web/interfaces/{holdem,table,server}.ts` from them;
  CI fails if they're stale. `web/interfaces/index.ts` adds the event union by hand.
- Outbound is mostly `table-update {view, events}` per viewer; `welcome` has the backlog on join.
- New inbound actions need a command in `table/commands.go`, mapping and validation in
  `server/client.go` and `server/validate.go`, a sender in `web/actions/actions.ts`, and tests.

## Conventions
- Commits: short, lowercase, imperative ("fix engine index panics").
