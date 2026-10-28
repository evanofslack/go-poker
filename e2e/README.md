# e2e

Browser tests that drive real players through the static build. They aren't run in CI, so they only
protect what you run them on.

| Script | Covers |
| --- | --- |
| `session.mjs` | hidden cards, next hand, reload and reconnect, turn timer and grace period |
| `betting.mjs` | betting controls: labels, presets, raise, the waiting line |
| `preaction.mjs` | picking an action before your turn: offers, clearing, playing it |
| `keys.mjs` | keyboard shortcuts, the raise panel's arrows, turning them off |
| `showdown.mjs` | action tags and the showdown display |
| `runit.mjs` | the run it vote, extra boards and their history |
| `houserules.mjs` | antes, the 7-2 setting, showing cards after the hand, duck hunting |
| `standup.mjs` | stand-up rounds: starting one, showing to sit down, the payout, running it again |
| `turn.mjs` | turn countdown |
| `seat.mjs` | seat menu: sit out and in, rebuy, sit out next hand, leave |
| `forms.mjs` | join, create table and buy-in forms |
| `password.mjs` | password tables and invite links |
| `players.mjs` | table panel: players and ledger |
| `host.mjs` | host controls: manage a player, chips, banker, remove, hand over host |
| `endsession.mjs` | host tab: pause, end session, settle-up |
| `approval.mjs` | rebuy approval |
| `stats.mjs` | table panel: stats tab |
| `history.mjs` | table panel: hands tab, hidden, shown and mucked cards, late joiners and reloads |
| `settings.mjs` | table panel: settings tab, for the host and a player; the host editing settings, mid-hand too |
| `options.mjs` | table panel: options tab, animations, deck, table and card backs, mute and shortcuts kept across reloads, old settings carried over |
| `sound.mjs` | sounds: the table, your turn, mute, a group off, a hidden tab, chat, test buttons |
| `ux.mjs` | toasts, reconnecting, the header, leaving |
| `layout.mjs` | nine seats on desktop, laptop, tablet and phone screens; six, eight and four players on phones |
| `bots.mjs` | adding, playing with and removing bots |
| `lobby.mjs` | public tables on the home page: listing, order, lock, joining from the list |
| `restart.mjs` | tables surviving a graceful restart mid-hand and a crash, on SQLite; runs its own server on :8097 |

## Setup

- `cd e2e && npm install`
- `npx playwright-core install chromium`, or point `CHROMIUM_PATH` at a Chromium you already have

## Running

- `just e2e` builds the web app and server, runs the server on :8099 and runs every script, about
  40s in all.
- `just e2e host stats` does the same for just those scripts.
- With a server already on :8099, run one script directly: `cd e2e && node host.mjs`. Rebuild
  `web/out` first if you changed the web app; the server serves the last build.
- `URL=<url>` points the scripts at another server. `SHOTS=<dir>` saves screenshots.

`just e2e` runs the server with short timers, `RECONNECT_GRACE=20s` and `BOT_DELAY=100ms`, and the
scripts read `RECONNECT_GRACE` to know how long to wait. Against a server with the defaults, set it
to `60s` (or leave it unset) for `session.mjs`.

`run.mjs` runs the scripts 4 at a time (`JOBS` changes it), the slowest first, and prints each
script's output in one piece when it finishes, then which failed and the slowest three. `lobby.mjs`
runs on its own at the end, since it checks the list of public tables.

Each script prints PASS or FAIL per check and a tally, and exits nonzero on any failure.
`session.mjs` is the slowest at about 35s: it waits for the 15s turn timer and the grace period.

## When to run what

- While working, run the scripts for the area you're changing. The table above says which.
- Before opening a PR that changes the web app, run `just e2e` once.
- Backend-only changes don't need e2e; `just check` covers them.
- For any visual change, save screenshots with `SHOTS` and look at them, at phone size too. Checks
  can't tell that a chart line is invisible or text is unreadable; a look can.

## Helpers

`lib.mjs` has what most scripts need, so a new one starts from a seated table:

```js
const browser = await launch();
const [alice, bob] = await seat(browser, "mytable" + Date.now(), ["alice", "bob"]);
await alice.click(btn("start"));
check("bob sees the waiting line", await has(bob, "text=waiting for alice"));
```

- Tables and players: `seat` (create, join and sit everyone; a player can be
  `{ name, seat, context }`, and a setup function adjusts the create dialog), or `open`, `create`,
  `join` and `sit` one at a time.
- Pace: pass `quick` as the setup for the 2s pause between hands, in scripts that play several and
  don't act between them. `grace` is the server's reconnect grace in ms.
- Waiting: `has` and `gone` for a selector, `until` for any condition, `quiet` for checks that
  something doesn't happen.
- Playing: `act` clicks whichever page has a control, `fold` folds for whoever's turn it is,
  `checkDown` checks or calls until a condition.
- The server's side: `record` keeps every view, log line and finished hand a page receives
  (call it before the page connects), `last` is the latest view.
- The UI: `btn` selects a button by its exact text (`button:text-is` misses Mantine's, which keep
  the label in a span), `panel` opens a table panel tab, `seatMenu` picks from your seat menu,
  `playerRow` reads a line of the players tab.

## Writing checks that don't flake

Most failures so far were timing mistakes in the test, not bugs in the app. In particular:

- Wait for the state you expect, never a fixed sleep: `has`, `gone` or `until`. Waiting for
  something not to happen is the exception, and uses `quiet` so it's easy to find.
- Only act on enabled controls (`button:text-is("fold"):enabled`). Both players see a fold button,
  but only the one whose turn it is can use it.
- An action can lose a race with the turn moving on. `act` retries with a short click timeout
  instead of one long wait.
- Frames already received still count: after a reload, wait for a new `welcome` (`rec.welcomes`),
  not for a view that was there before.
- Check one behaviour per check, at the level a user would notice. Exact text and pixel details are
  what break first.
- For layout checks, find elements by a `data-` attribute rather than their text, and add one if
  there isn't one, so markup changes don't break the lookup.
- Starting a server yourself? Check :8099 is free first (`just e2e` does) and stop it afterwards.
  A leftover server makes every script fail.
