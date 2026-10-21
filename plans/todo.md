# Roadmap

| Stage | Theme | Status |
|---|---|---|
| 1 | Security | done |
| 2 | Game correctness and cheating | done |
| 3 | Upgrades | done |
| 4 | Core play experience | done |
| 5 | Host and table features | planned |
| 6 | UI consistency and mobile | done |
| 7 | Operator tooling | planned |
| - | Backlog | unranked |

## How items are ranked

1. **Trust:** can a player cheat or fake something?
2. **Table-breaking:** can one person or edge case stall or corrupt the game?
3. **Core loop clarity:** can a new player tell what to do and what happened?
4. **Host control:** can someone run a home game without friction?
5. **Polish and reach:** consistency, mobile, accessibility.
6. **Operating at scale:** matters once the public demo gets real traffic.

Within a tier, sequence work so earlier PRs don't get rewritten: server plumbing (identity, errors, timers)
before the UI that uses it, and design tokens before big component rewrites.

---

# Stage 2: game correctness and cheating

Roughly in priority order. Split into small PRs: one per section is a reasonable unit.

## 2.1 Hidden information (cheating)

- [x] **Stop sending every player's hole cards to every client.**
  `createUpdatedGame` (`backend/server/events.go`) broadcasts `GenerateOmniView()`, which hides nothing.
  The frontend only hides other players' cards visually (`web/components/Seat.tsx`), so DevTools shows every hand.
  - Send each client its own view via `GeneratePlayerView(pn)` (`backend/poker/views.go`), which already exists.
  - Spectators get a view with all hole cards hidden.
  - This means per-client sends instead of one table broadcast, so `table.broadcast` needs a per-client
    variant (or the table builds a view per client when it sends).
  - Check `GeneratePlayerView`'s showdown logic (it reveals winners' and called hands at `PreDeal`).
  - Decided: reveal every hand that went to showdown (no mucking for now).

## 2.2 Identity: act only as your own seat (cheating)

- [x] **Link each connection to its player and reject actions out of turn.**
  `handleCall`/`handleRaise`/`handleCheck`/`handleFold` act for `view.ActionNum`, i.e. whoever's turn it is,
  so any client (even a spectator) can bet or fold for someone else.
  - `c.uuid` is already set to the player's UUID on take-seat. Look up the player index by UUID
    (indexes change when `setSeatID` re-sorts players) and require it to equal `ActionNum`.
  - Spectators (connected but not seated) can't send any game action. Cover this explicitly in server tests.
- [x] **Host role** for start/deal/reset (currently any seated player can reset the table).
  Decided: the player who creates the table is host. Split create-table from join-table.
  Start and deal don't need the host; reset does.
- [x] **Reserved names:** reject `system` (`gameAdminName`) and names already used at the table, so nobody
  can impersonate the server or another player in chat. Decided: case-insensitive, spectators included.
- [x] **Chat limits:** cap message length and rate per client server-side. (#19)
- [x] Chat uses the server-side username, not one from the message. (#19)
- [x] Optional: let a player choose a username once per connection (`new-player` can be re-sent today).

## 2.3 Server-owned log and hand lifecycle (cheating)

- [x] **Remove client `send-log`.** The browser writes the game log ("alice bets 50") and the server relays it,
  so the log can be faked. Generate log lines in the server handlers after an action succeeds instead.
  - Client call sites to remove: `RaiseInput.tsx`, `Table.tsx`, `BuyIn.tsx`, `Input.tsx` (call/check/fold).
  - Drop `actionSendLog` from `backend/server/client.go` and `messages.go`.
- [x] **Server drives the next hand.** Every browser runs its own 5s timer after showdown and sends
  `deal-game`, and every browser sends the "X wins N" log (`web/components/Table.tsx`), so a 6-player table
  gets 6 deals and 6 win logs. Move the showdown pause and auto-deal to a timer on the table goroutine;
  the client just renders. The turn timer in 2.4 uses the same mechanism.
- [x] **Report action errors back to the client** as a typed event (seat taken, bad buy-in, not your turn,
  invalid bet). Today a rejected action is only logged on the server.

## 2.4 Disconnects, leaving, and reconnecting

Moved up from last place: a closed tab breaks almost every casual session. Depends on 2.2 (identity) and
2.3 (server timers).

- [x] **A closed tab blocks the table forever.** Disconnect removes the client but the player stays in the game,
  and the hand waits on them. Nothing calls `poker.Leave`.
  - On disconnect/leave-table: fold if it's their turn or they're in the hand, then mark them left/not-ready.
  - Remove left players' seats between hands.
- [x] **Turn timer** so an idle player can't stall the table. Decided: 30s, auto-check if possible, else fold.
- [x] **Reconnect.** Issue a session token on take-seat; the client stores it and re-presents it on reconnect
  to reclaim the seat within a grace period. Design together with 2.2's connection-to-player link.
  Decided: 60s grace period, token in sessionStorage.
- [x] Client auto-reconnects with backoff (`web/providers/WebSocket.tsx`) and rejoins its table.

## 2.5 Engine bugs (confirmed with tests, not yet fixed)

- [x] **A short stack can't call an all-in for less.** `bet()` (`backend/poker/actions.go`) rejects
  `betVal < minBet - p.Bet` even when `betVal` is the player's whole stack. The frontend caps the call at
  the stack, so the player is stuck and can only fold. Allow a bet equal to the remaining stack (all-in) for
  less than the call, and for less than a min-raise.
- [x] **Busted players stay `In`.** `resetForNextHand` (`backend/poker/game.go`) marks players with no chips
  as not ready but leaves `In = true`, and `deal` only sets `In` for ready players. Busted players then count
  toward `inPlayerNums`, which affects when a hand ends. Reset `In` (and `Called`) for everyone at the end of a hand.
- [x] **Sitting down mid-hand reorders players.** `setSeatID` sorts `g.players` by seat, so `actionNum`,
  `dealerNum`, `sbNum`, `bbNum` now point at different people. Only allow seating between hands
  (or seat new players as not-ready and re-sort only at `PreDeal`). Decided: sit out until the next hand.
- [x] **Split pots lose odd chips** (`updateRoundInfo`: "fractional money will disappear"). Give the
  remainder to the first winner left of the dealer.
- [x] **Infinite loop guard** (from 2.6, cheap enough to do here): `resetForNextHand` and `updateBlindNums`
  loop `for !players[i].Ready` with no exit, so they hang forever if nobody is ready (holding the table lock).
  Guard like `toggleReady` now does.

## 2.6 Engine robustness (from reading the code, not yet proven)

- [x] Everyone all-in but one: betting still opens each street (TODO in `deal`). Run the board out instead.
- [x] `handleCall` (`events.go`) computes the call from `TotalBet` (whole hand) while `bet()` uses `Bet`
  (this street). Equivalent in practice today, but use the same source of truth.
- [x] `Start()` refuses if any seated player isn't ready. A not-ready player should just sit the hand out.

## 2.7 Frontend fixes noted along the way

- [x] **Buy-in config name mismatch:** frontend reads `config.maxBuyIn` (`web/components/BuyIn.tsx`,
  `web/interfaces/index.ts`), server sends `maxBuy`. Fixing it changes the default buy-in from 2000 to the
  10000 cap, so pick a sensible default at the same time. Decided: 100 big blinds.
- [x] **Hooks after an early return** in `RaiseInput.tsx` (`useState`/`useCallback` after
  `if (!appState.game) return null`). Breaks React's rules of hooks; move the hooks above the return.
  Also in `Pot.tsx`.
- [x] **Websocket cleanup is a no-op** in `web/providers/WebSocket.tsx`: the effect cleanup closes a stale
  `socket` (always null), so strict mode in dev opens two sockets. Close `_socket` instead.
- [x] Show server-side errors from 2.3 to the user (seat taken, bad buy-in, not your turn).
- [x] `npm run start` (`next start`) doesn't work with `output: 'export'`; remove the script.

## Stage 2 testing

- Add engine regression tests for each bug in 2.5 and 2.6 (failing first, like the stage 1 probes).
- Add server tests for 2.1-2.4: a client can't see other hands, can't act out of turn or as a spectator,
  can't send logs, only one deal happens per hand, and a disconnect mid-hand doesn't stall the table.
- The two-player headless Chrome script used in stage 3 (join, sit, start, bet, chat with screenshots) is a
  good end-to-end check to keep around.

---

# Stage 4: core play experience

Makes a session pleasant. Do after 2.3 (error events) and 2.5 (all-in fix), or the action bar gets redone.
Consider the light token pass from 6.1 first so new components aren't restyled twice.

- [x] **Action bar redesign** (`web/components/Input.tsx`).
  - One context-sensitive check/call button; `canCall` is currently inverted from its name.
  - Label "bet" vs "raise" correctly depending on whether there's a bet to face.
  - Show the to-call amount clearly.
- [x] **Raise input** (`web/components/RaiseInput.tsx`): numeric input plus slider, min / ½ pot / pot / all-in
  presets, visible clamp feedback. Pot math should use all pots, not just `pots[0]`.
- [x] **Showdown display:** winning hand name ("two pair, kings and sevens"), highlight the winning cards,
  show split and side pots and each pot's winner. `winningHand`/`winningScore` are already sent;
  `getWinner` only reads `winningPlayerNums[0]` of the last pot.
  Done with the category only ("two pair"); naming the ranks would need the server to send them.
- [x] **Table markers:** SB/BB chips next to the dealer button.
- [x] **Turn notification:** sound and/or tab title flash when it's your turn; show the 2.4 turn timer
  as a countdown on the active seat. The server doesn't send the turn deadline yet; add it to `update-game`.
- [x] **Buy-in UX** (`web/components/BuyIn.tsx`): show min/max, validate, a labeled confirm button instead of
  a lone checkmark.
- [x] **Leave seat / sit out / rebuy** buttons (uses 2.4 engine work).
- [x] **Join form** (`web/components/Join.tsx`): one `autoFocus`, use a real `<form onSubmit>` instead of a
  document-wide `keydown` listener. (Create/join buttons and disabling are done.)
- The dealer button passing on when the dealer sits out is the engine's intended behavior; left as is.

# Stage 5: host and table features

Needs the host role from 2.2.

- [x] **Invite links:** a `/t/:name` route (or `?table=`) and a "copy invite link" button, instead of telling
  friends to type the same table name.
- [x] **Table settings** for the host: blinds, min/max buy-in, seat count, turn timer length,
  private/password tables. See 5.1.
- [ ] **Seat management:** kick a player, sit a player out, optionally approve buy-ins/rebuys.
- [ ] **Pause/resume** the game.
- [x] **Confirm before reset.** Reset now sits in a host menu and asks first.
- [ ] **Public tables:** list open tables on the home page so anyone can pick one to join. Needs a hub
  listing of tables (name, stakes, seats taken), and goes with private/password tables.
- [ ] **End session with a ledger:** net result per player (buy-ins vs final stack), copyable.
- [ ] **Spectator mode** made explicit in the UI. The header now counts spectators ("watching 2");
  a list of who's watching is left.

## 5.1 Table settings

The host picks settings in a dialog when creating a table. Settings are fixed once the table exists.

Today: `poker.NewGame()` hard-codes 10/20 and a 10000 max buy-in with no minimum. `maxSeats`,
`nextHandDelay` and `turnTimeout` are package constants in `backend/server/events.go`, and
`web/components/Countdown.tsx` hard-codes 30s.

| Setting | Default | Allowed | Form |
|---|---|---|---|
| Seats | 8 | 2-9 | presets 2 / 6 / 8 / 9 |
| Stakes (SB/BB) | 10/20 | 1 ≤ SB ≤ BB | presets 1/2, 5/10, 10/20, 25/50, 100/200, or custom |
| Min buy-in | 50 BB | ≥ 1 BB, ≤ max | in BB, chip amount shown beside it |
| Max buy-in | 200 BB | ≤ 1000 BB | in BB, chip amount shown beside it |
| Turn timer | 30s | off, 15, 30, 60, 120s | segmented control |
| Next-hand delay | 5s | 2, 3, 5, 8, 10, 15s | segmented control |
| Public | off | bool | toggle |
| Password | none | ≤ 32 chars | optional text field |

Decided:
- One delay after every hand, used for showdowns and folds alike.
- Public is stored and sent to clients but does nothing yet; the public table list (below) uses it later.
- Buy-ins are entered in big blinds and stored in chips.
- Seats cap at 9. Check the oval layout at 9 on mobile.
- Turn timer off means an idle connected player can stall the hand. Fine, the host chose it.
- No Mantine: native inputs, the existing `Button`, and a small segmented control component.

### PR 1: add table settings (server + protocol)

- [x] `tableSettings` on `table` with the fields above. `poker.NewGame(cfg)` takes the engine config.
  Per-table seats, turn timeout and next-hand delay replace the constants (tests still shorten them).
- [x] `create-table` takes an optional `settings` object; missing fields get defaults, so old clients
  and the e2e scripts keep working. Bounds checks in `validate.go`.
- [x] Min buy-in enforced in `handleTakeSeat`, and on rebuy when busted (topping up a stack with chips
  left can be any amount up to max).
- [x] Turn timer off: `startTurnTimer` does nothing, `turnEndsIn` stays 0 (the countdown already hides).
- [x] `update-game` sends the settings (`hasPassword`, never the password). `Countdown` reads the turn
  length from it; seat `BuyIn` uses min/max from it.
- [x] Tests: validation bounds, defaults, seat count and min buy-in enforced, no auto-fold with timer off.

### PR 2: add table passwords

- [x] `join-table` takes `password`. Compare in constant time; kept in memory only, never logged or sent.
- [x] A valid reconnect token skips the password, so a dropped player reclaims their seat.
- [x] Add a `code` to the `error` message (`password-required`, `wrong-password`) so the client can show
  a password field without matching on text.
- [x] Invite links put the password in the fragment: `?table=foo#pw=…`. The fragment is never sent in
  HTTP requests or `Referer`, so it stays out of server and proxy logs. The client reads it, strips it
  from the address bar with `history.replaceState`, and sends it over the websocket (wss in prod).
  Anyone with the link can join, which is the point; it stays in the sharer's clipboard and chat history.
- [x] Tests: no password, wrong password, right password, reconnect without password.

### PR 3: add create table form

- [x] "create" on the join form opens a settings dialog (name and table already entered). Submitting
  sends `create-table`; the table doesn't exist until then. Errors (name taken, invalid value) show in
  the dialog and keep it open. On `joined-table` the host lands at the table as today.
- [x] Defaults work with no edits. Seats, timer and delay go under a collapsed "more" section.
- [x] `Dialog` gets a width option (fixed `w-72` today). Settings presets live in `web/lib/settings.ts`.
- [x] Update `e2e/forms.mjs` for the dialog; check the dialog and a 9-seat table at phone width.

Done in #56, #57, #58. Differences from the plan:
- The server takes any value in range, not just the presets: turn timer off or 10s-5min, delay 2-15s,
  big blind up to 1,000,000. Settings a client leaves out entirely are the trusted defaults.
- `joined-table` sends the password back to the client that joined, so every member can copy a
  working invite link.
- `layout.mjs` now fills a 9-seat table on every viewport. On a 360px phone the SB marker is partly
  covered by the next seat's cards; seats themselves don't overlap.
- Not done: a per-table limit on wrong password attempts (only the per-connection rate limit applies).

# Stage 6: UI consistency and mobile

## 6.1 Design tokens (light pass, can go before Stage 4)

- [x] Pick one neutral palette (currently `zinc`, `neutral`, and `gray` are mixed), a radius scale
  (currently `rounded-sm`/`xl`/`2xl` drift), and spacing. Put them in the CSS `@theme` (Tailwind 4, no config file).
- [x] Button variants: primary, secondary, danger, ghost. Replace ad-hoc button classes.
- [x] One voice for labels (currently lowercase "join"/"fold" but "Start").

## 6.2 Feedback states

- [x] Connection status indicator (connected / reconnecting / disconnected).
- [x] Toasts for server errors (reusing 2.3/2.7 plumbing).
- [x] Loading and empty states (new table shows six "Open" seats with no guidance).

## 6.3 Layout and accessibility

- [x] Responsive layout. Today it's absolutely positioned corners, fixed `w-96` chat and `w-56` seats,
  and `GameInfo` hidden on small screens.
- [x] Compute seat positions around an oval from seat count instead of hand-placed `chipPosition`
  percentages (also unlocks 9-max tables from Stage 5 settings).
- [x] Simplify `Seat.tsx` (four render branches, duplicated markup).
- [x] Accessibility: labels for cards and buttons, turn state not conveyed only by glow/color,
  visible focus styles.

# Stage 7: operator tooling

Matters once the public demo gets real traffic.

- [ ] **Health/metrics:** proper health and readiness endpoints (`/ping` returns the JSON string
  `"message: pong"`), Prometheus metrics (tables, clients, hands/min, dropped slow clients, rate-limit hits).
- [ ] **Admin view:** a protected endpoint/page listing tables, player counts, connections, uptime; close a
  table or kick a connection.
- [ ] **Lifecycle:** reap idle tables; on graceful shutdown, notify clients ("server restarting") before closing.
- [ ] **Config:** move hard-coded limits (`maxTables`, `maxSeats`, rate limits, timers) to env vars and document
  them in `.env.example`.
- [ ] **Audit log:** structured per-table logs of actions for debugging disputes.
- [ ] Note in the UI and README that state is in-memory and lost on restart.

# Backlog (unranked)

- Hand history / review last hand
- Pre-action buttons (check/fold, call any) and keyboard shortcuts
- Run it twice, straddles, antes, show/muck at showdown
- Pot-limit Omaha
- Sit-n-go / tournament mode with a blind schedule
- Accounts and persistent stats (only if this should be more than a play-with-friends app)
