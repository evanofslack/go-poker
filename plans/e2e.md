# e2e cleanup

**Status: PRs 1 and 2 done; CI next.** Stage 10 in `todo.md`, before straddles and run it (`straddle.md`).

The goal: a suite that's quick enough to run before every web PR without thinking about it, that
doesn't flake, and where a new script is short. Stage 11 adds the most timing-heavy scripts yet
(the straddle prompts, the run-it vote), so they should be written against the new helpers.

## Where it stands (2026-10-26)
- 23 scripts, about 2150 lines, 607 checks. A full `just e2e` passed in 4m08s.
- Time per script: `session` 72s, `bots` 51s, `layout` 16s, `stats` 14s, `history` 13s, the rest
  1-8s each. `session` and `bots` are half the run, waiting on real timers: the 60s reconnect
  grace, the 30s turn timer, bots thinking 1-2s a move, 5s between hands.
- Scripts run one after another, each with its own browser.
- The same code is copied around:
  - the create, join, sit loop: 22 scripts;
  - `record` (every frame a page receives) and `waitFor`: `session`, `bots`, `restart`;
  - `enter` and `sit`: `session`, `restart`, `approval`, `lobby`;
  - check or call down to showdown: `history`, `showdown`; fold a hand: `stats`;
  - `has` (wait for a selector, as a bool): `keys`, `history` and others, each written inline.
- About 30 fixed `waitForTimeout` sleeps. Twelve are the 300ms after sitting in the copied loop;
  the rest wait for a turn to move, a street to deal, or prove nothing happened.

## Decisions
- Keep plain node scripts on `playwright-core`, no test runner. They're readable and `check`
  output is enough; a runner would add a dependency and a style to learn for little.
- One server for every script, as now, except `restart`, which runs its own.
- Behaviour the user notices is what's checked. Exact text and pixels only where they are the
  feature (layout overlap, the countdown bar).

## PR 1: shared helpers
New in `lib.mjs`:

```js
// seat creates a table and seats each player in order, returning their pages.
// settings are applied in the create dialog (e.g. { turnTime: 10, nextHandDelay: 2 }).
const [alice, bob] = await seat(browser, ["alice", "bob"], { settings, seats: [1, 2] });

record(page)                     // { games, logs, hands, errors }, every frame received
waitFor(cond, ms, what)          // poll a condition; false and a log line on timeout
has(page, selector, ms = 3000)   // wait for a selector, as a bool for check
act(page, label)                 // click an enabled control, retrying while the turn moves on
checkDown(pages, done)           // check or call until done() (the showdown, the next hand)
foldHand(pages)                  // whoever's turn it is folds, until the hand ends
quiet(page, ms)                  // a named wait for checks that nothing happens
```

- `seat` waits for the seat to show as taken instead of sleeping 300ms.
- Move every script onto them. Scripts that test the forms themselves (`forms`, `password`,
  `lobby`) keep driving the forms by hand.
- Replace each fixed sleep with a wait for what it stands for: an enabled button, a frame in
  `record`, a log line. The "nothing happens" ones (`keys`: shortcuts turned off, typing in chat)
  use `quiet`, so they're easy to find. Animation waits become `settle`, also named.
- Trim checks that pin exact text where the behaviour is what matters (button label lists in
  `betting`, the pot text regex in `history`, the won label regex in `showdown`).
- Expected: about a third fewer lines.

## PR 2: faster
- Short timers for e2e. `just e2e` starts the server with `RECONNECT_GRACE=5s` (already an env)
  and a new `BOT_DELAY=100ms` (`envDuration` for `Timing.BotDelay` in `server/config.go`).
- `session` creates its table with the 10s turn timer (the minimum) and reads the grace from
  `RECONNECT_GRACE`, so it waits about 15s instead of 90s.
- Scripts that play several hands (`stats`, `history`, `bots`, `standup`) create with the 2s next
  hand delay.
- Run scripts in parallel, 4 at a time, in `just e2e` (a small `e2e/run.mjs` that spawns each
  script and prints its output in one piece when it finishes, then the tally of scripts and the
  slowest). Raise `MAX_CONNS_PER_IP` for the e2e server, since every browser connects from one
  address. `lobby` runs alone after the rest, since it checks the public table list; `restart`
  already runs its own server on :8097.
- Print each script's time and the total.
- Target: under a minute for the whole suite, from about four.

Differences from the plan:
- `RECONNECT_GRACE=20s`, not 5s: the turn timer has to run out before the grace does, or bob is
  folded for leaving instead of timing out. The create form's shortest turn timer is 15s, not 10s.
- `standup` keeps the 5s pause between hands, since it acts between them.
- The suite takes about 40s (37s of scripts); `session` is the long pole at 35s.

## PR 3: CI
- A CI job, `e2e`, on PRs that touch `web/`, `e2e/` or `backend/`: build the web app and server,
  install Chromium (cached), run `just e2e`, upload screenshots and output when it fails.
- Non-blocking at first (`continue-on-error`); blocking once it's been green for a couple of
  weeks of PRs.
- Update `e2e/README.md`: the helpers, the short timers, parallel runs, CI. Update CLAUDE.md's
  e2e line if CI runs it.

## Checking it worked
- Run the whole suite 5 times in a row after PR 2; every run passes.
- `git diff --stat` on PR 1 shows the scripts shrinking, and no check removed without a reason in
  the PR.
- The time per script is printed, so a slow new script stands out.
