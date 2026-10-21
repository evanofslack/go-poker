# e2e

Browser tests that drive real players through the static build. They aren't run in CI, so they only
protect what you run them on.

| Script | Covers |
| --- | --- |
| `session.mjs` | hidden cards, next hand, reload and reconnect, turn timer and grace period |
| `betting.mjs` | betting controls: labels, presets, raise, the waiting line |
| `showdown.mjs` | showdown display |
| `turn.mjs` | turn countdown |
| `seat.mjs` | seat menu: sit out and in, rebuy, sit out next hand, leave |
| `forms.mjs` | join, create table and buy-in forms |
| `password.mjs` | password tables and invite links |
| `players.mjs` | table panel: players and ledger |
| `host.mjs` | host controls: manage a player, chips, banker, remove, hand over host |
| `endsession.mjs` | host tab: pause, end session, settle-up |
| `approval.mjs` | rebuy approval |
| `stats.mjs` | table panel: stats tab |
| `ux.mjs` | toasts, reconnecting, the header, leaving |
| `layout.mjs` | nine seats on desktop, laptop, tablet and phone screens |
| `bots.mjs` | adding, playing with and removing bots |
| `lobby.mjs` | public tables on the home page: listing, order, lock, joining from the list |

## Setup

- `cd e2e && npm install`
- `npx playwright-core install chromium`, or point `CHROMIUM_PATH` at a Chromium you already have

## Running

- `just e2e` builds the web app and server, runs the server on :8099 and runs every script.
- `just e2e host stats` does the same for just those scripts.
- With a server already on :8099, run one script directly: `cd e2e && node host.mjs`. Rebuild
  `web/out` first if you changed the web app; the server serves the last build.
- `URL=<url>` points the scripts at another server. `SHOTS=<dir>` saves screenshots.

Each script prints PASS or FAIL per check and a tally, and exits nonzero on any failure.
`session.mjs` takes about 100s, since it waits on the real 30s turn timer and 60s grace period.

## When to run what

- While working, run the scripts for the area you're changing. The table above says which.
- Before opening a PR that changes the web app, run `just e2e` once.
- Backend-only changes don't need e2e; `just check` covers them.
- For any visual change, save screenshots with `SHOTS` and look at them, at phone size too. Checks
  can't tell that a chart line is invisible or text is unreadable; a look can.

## Writing checks that don't flake

Most failures so far were timing mistakes in the test, not bugs in the app. In particular:

- Wait for the state you expect, never a fixed sleep. Use `waitForSelector`, or poll for a condition
  with a timeout.
- Only act on enabled controls (`button:text-is("fold"):enabled`). Both players see a fold button,
  but only the one whose turn it is can use it.
- An action can lose a race with the turn moving on. Retry with a short click timeout instead of one
  long wait.
- Check one behaviour per check, at the level a user would notice. Exact text and pixel details are
  what break first.
- For layout checks, find elements by a `data-` attribute rather than their text, and add one if
  there isn't one, so markup changes don't break the lookup.
- Starting a server yourself? Check :8099 is free first (`just e2e` does) and stop it afterwards.
  A leftover server makes every script fail.
