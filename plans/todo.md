# Roadmap

| Stage | Theme | Status |
|---|---|---|
| 1 | Security | done |
| 2 | Game correctness and cheating | done |
| 3 | Upgrades | done |
| 4 | Core play experience | done |
| 5 | Host and table features | done |
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
- [x] **Seat management:** kick a player, sit a player out, optionally approve buy-ins/rebuys. See 5.3.
- [x] **Pause/resume** the game. See 5.3.
- [x] **Confirm before reset.** Reset now sits in a host menu and asks first. 5.3 removes reset.
- [x] **Public tables:** list open tables on the home page so anyone can pick one to join. The join form
  polls `list-tables` every 5s; password tables show a lock, most players first, full tables last.
- [x] **End session with a ledger:** net result per player (buy-ins vs final stack), copyable. See 5.3.
- [x] **Spectator mode** made explicit in the UI. The header counts spectators ("watching 2") and the
  5.3 players panel lists them.
- [x] **Bots** so one person can demo the app alone (easy and medium levels). See 5.2.

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
- Public tables are listed on the home page. The create form defaults to public; a create-table without
  settings stays private.
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

## 5.2 Bots

The host fills empty seats with computer players, so one person can play a full table. No off-the-shelf
bot fits a Go server, so they're hand-written on top of the engine's evaluator (`riverboat/eval`).
Levels share one decision function and differ in its inputs and settings, so they're cheap to add.

Decided:
- Bots are game players with no `Client`. The table keeps `bots map[string]string` (uuid to level),
  guarded by `mu`. They don't count as members, so a table with only bots left is deleted as today.
- Only the host adds or removes bots. Adding follows the same seat and timing rules as `take-seat`.
- A bot decides from `GeneratePlayerView(pn)` only, so it never sees anyone else's cards.
- "Add bot" opens a dialog like sitting down: buy-in (defaults to the same 100 BB as a player's,
  within the table's min/max) and a name. Bots rebuy for that same amount when busted.
- The name is prefilled with a random first name of a character from HBO's *Industry* that isn't
  taken at the table (Harper, Eric, Yasmin, Robert, Gus, Rishi, Venetia, Kenny, Daria, Sweetpea,
  Henry, Jesse), and the host can edit it. The list lives in the web client; the server only checks
  the name like a username (length, reserved, not taken). The name is fixed once the bot sits.
- The UI marks bots with a badge, separate from the name.
- A game can run with only bots seated: the host can start it without a seat, and the server keeps
  dealing. A table with no connected clients is still deleted, bots and all.
- Bots keep playing while their table waits out a disconnected player's reconnect grace. Fine.

### PR 1: add easy bots

Built off main alongside #62 (ledger) and #63 (players panel). Done when those were rebased onto it:
bot buy-ins and auto-rebuys go in the ledger, and the players panel shows the bot badge.


- [x] `backend/bot/`: `Decide(level, view, pn, rng) Action` (check/call/raise/fold plus amount), pure
  and table-free so levels are unit tested on their own.
- [x] **Easy** (calling station): checks when free, calls most bets, folds weak hands to big bets
  (over ~⅓ of its stack), and min-raises now and then with a pair or better. Some randomness so it
  isn't fully predictable. Uses the evaluator for its made hand, no simulation.
- [x] Turn taking: when `startTurnTimer` lands on a bot, schedule its move after a short random delay
  (about 1-2s, a var so tests shorten it) tied to `turnSeq`, so a reset or turn change cancels it.
  The move checks the table isn't done and the bot is still the player to act.
- [x] Pull call/check/raise/fold out of the client handlers into `act(t, pn, kind, amount)`, shared by
  clients, bots and `turnTimedOut`. If a bot's action is rejected, it checks or folds instead.
- [x] Protocol: inbound `add-bot {seatID, buyIn, name, level}` and `remove-bot {seatID}`; `update-game`
  gets `bots: uuid[]`. Update `messages.go`, `actions.ts`, `interfaces/index.ts`; validate name and
  `level` in `validate.go` (only `easy` for now). Buy-in uses the same bounds as `take-seat`.
- [x] Share the seating steps (add player, name, buy in, ready, seat) between `handleTakeSeat` and
  add-bot instead of copying them.
- [x] `start-game` is allowed for the host even without a seat, so a bots-only game can start.
- [x] Remove a bot like a leaving player: folded on its turn and removed after the hand.
- [x] Web: the host sees an "add bot" button on empty seats (seated or not, mid-game too) that opens the
  buy-in and name dialog. Bot seats show a "bot" badge, and the host gets a remove button on them.
  Bot names in `web/lib/bots.ts`. No level picker until there's a second level.
- [x] Tests: host-only add/remove, seat taken, bad buy-in, name taken or invalid, bot acts on its turn
  and the hand finishes, bot can't see hole cards, busted bot rebuys for its buy-in, host starts a
  bots-only game and it keeps dealing, table with only bots and no clients is deleted, level validation.
  `e2e/bots.mjs`: host sits, adds two bots, starts, and plays a hand to showdown.

Differences from the plan:
- Bots think 1-2s per move, so a heads-up hand of only bots takes about 15s.
- "Bot can't see hole cards" is checked in `e2e/bots.mjs` (a spectating host sees no bot cards
  mid-hand). On the server it holds because `bot.Decide` only gets `GeneratePlayerView(pn)`.
- `bot.Decide` reduces the view to a small `spot` struct, since the engine's player type isn't
  exported, so tests build spots directly.

### PR 2: add medium bot

- [x] **Equity:** Monte Carlo against random hands (a few thousand runouts, a few ms with the
  evaluator), one random hand per opponent still in.
- [x] **Pot odds:** call when equity beats the price of calling, fold otherwise; raise for value when
  equity is well above it. Bet sizes of ½ to ¾ pot instead of min-raises.
- [x] **Preflop:** a starting hand chart (or the Chen formula) instead of equity against random hands,
  which overrates junk at full tables.
- [x] Small bluff and slow-play rates so it isn't purely mechanical.
- [x] Level picker on "add bot" (easy / medium).

Differences from the plan:
- Preflop uses the Chen formula (raise at 10+, call a raise at 7+, limp at 5+, two points looser heads
  up). Postflop runs 1000 runouts against random hands, about 2.4ms against 8 opponents.
- The add-bot dialog defaults to medium. `update-game` sends `bots` as `{uuid: level}` so seats read
  "easy bot" or "medium bot".
- `backend/bot/sim_test.go` plays bots against each other on the real engine (`BOTSIM=1`). Medium beats
  easy by about +120 to +165 big blinds per 100 hands heads up.

### Later: hard bot

- [ ] **Position:** tighter ranges early, wider on the button and in the blinds.
- [ ] **Ranges, not random hands:** run equity against the hands an opponent likely holds given how they
  bet this hand (raised preflop: strong range; called: medium).
- [ ] **Sizing:** pick bet sizes by board texture and stack depth; shove short stacks preflop.
- [ ] **Balanced bluffing:** bluff with draws and blockers at a frequency matched to bet size, so it's
  hard to exploit by always calling or always folding.
- [ ] **Light opponent modelling:** track each player's VPIP/aggression at the table and adjust (bluff
  tight players more, value bet loose ones).
- [ ] Optional personalities on the same settings (tight-passive, loose-aggressive) for variety.
- [ ] Check that hard beats medium in `bot/sim_test.go` (`BOTSIM=1 go test ./bot -run Sim -v`).

## 5.3 Ledger, players panel and host controls

A ledger of who bought in and cashed out, settled through a banker; a players panel everyone can see;
and host controls to run a home game without a reset button.

Today: the only chip history is the engine's `TotalBuyIn`, which is lost when a player leaves their seat
(`RemovePlayer` deletes them) or the host resets. A table is deleted as soon as it's empty. The host is a
connection (`t.host *Client`), so a dropped connection hands it to someone else right away, even though
the host's seat is kept for the reconnect grace period. Reset is the host menu's only item.

Decided:
- Chips only, no money conversion.
- Everyone at the table sees the ledger, spectators included.
- Ledger rows are by name. Someone who leaves and comes back under the same name continues their row,
  and so would anyone else who takes that name later. Accepted for friends' tables; log "bob rejoined".
- Settle up through a banker the host picks (the host by default, changeable any time). Each player's
  line is against the banker: up means "request X from the banker", down means "send X to the banker".
- The host role belongs to a player, not a connection, and the host can hand it to another player.
- Remove reset. Kick, chip edits, end session and making a new table cover every use of it.
- Rebuy approval is a table setting, off by default.
- End session only takes effect after the current hand.
- An ended table stays readable for 1 hour after it ends, so the invite link still shows the ledger.

### PR 1: add ledger (server)

- [x] Per-table ledger entries: time, name, kind (buy-in, rebuy, cash-out; host add and remove come
  with PR 3), amount, and who did it. Totals come from the entries.
- [x] Record entries where chips enter or leave: `handleTakeSeat` (buy-in), `handleRebuy`, and
  `removeLeavers` for cash-outs (leaving a seat, leaving the table and grace expiry all end there).
  Until PR 3 removes it, reset cashes everyone out, counting this hand's bets.
- [x] Rows per name: bought in, cashed out, current stack, net. Mid-hand, chips bet this hand still
  count as the player's, so net is live during a hand (and the balance stays zero mid-hand too).
- [x] Banker: the host until PR 3 lets the host pick. The client works out each banker line from the
  row's net, so it isn't sent.
- [x] Balance check: the sum of nets, sent so the panel can show "balanced" or "off by N".
- [x] `update-game` sends the rows, the banker, the balance and the names of spectators (`watchers`
  replaces the `watching` count).
- [x] Tests: buy-ins, rebuys, cash-out on leaving a seat, leaving and coming back by name, grace expiry,
  live nets mid-hand, reset. Split pots aren't tested separately; the engine already gives odd chips out.
- Moved to PR 5: stack snapshots per hand. State is in memory and a deploy restarts the server, so
  recording before the chart ships keeps nothing.

### PR 2: add players panel

- [x] A "players" tab beside chat and log: every name with status (playing, sitting out,
  disconnected, left, watching), seat, stack, bought in, net, and their banker line.
- [x] Shows the banker, the balance check, and a "copy ledger" button with plain text for a group chat.
- [x] e2e: rows update on buy-in, rebuy and leaving; copy text; phone layout (`players.mjs`).
- Also: `update-game` sends `disconnected`, the names of seated players in their reconnect grace,
  so the panel can tell disconnected from playing. "Out of chips" is its own status.

### PR 3: add host controls

- [x] Host is a name on the table. It passes on only when the host leaves for good: leaves the table,
  is removed, or their reconnect grace runs out. Reconnecting within grace keeps it. A spectator host
  has no seat to hold, so it passes on when they disconnect.
- [x] Host actions, each host-only, validated in `validate.go`, logged to the table, and tested:
  - make another player host
  - pick the banker
  - move a player to spectator, or remove them from the table (mid-hand they're folded on their turn,
    like leaving). No bans.
  - sit a player out (mid-hand it applies after the hand)
  - add or remove chips, between hands only, recorded as a host add or host remove with a reason
- [x] Per-player host actions live in a menu on each row of the players panel.
- [x] Remove reset: `reset-game`, `handleResetGame`, the host menu item, its tests and the e2e uses
  (`ux.mjs`, `session.mjs`, `betting.mjs`).

Notes:
- `update-game` sends the host's name instead of a uuid.
- Removing tells the player's client to leave (error code `removed`); it leaves, forgets its session and
  shows why on the join form. The seat is taken server-side either way.
- Host chip adds still respect the max buy-in. The host can't remove themselves.
- The host menu went with reset, since reset was its only item. PR 4 brings it back for pause and
  end session. e2e: `host.mjs`.

### PR 4: add session controls

- [x] Pause and resume. Pausing mid-hand takes effect after the hand; no new hand is dealt while
  paused. Logged.
- [x] End session, after the current hand. Dealing stops, seats and rebuys close, the ledger is final,
  and everyone sees the final settle-up. The table is deleted 1 hour after it ends, empty or not;
  until then joining it (with its password) shows the final ledger.
- [x] Rebuy approval setting (`approveRebuys`, off by default) in `tableSettings` and the create form.
  When on, a rebuy waits for the host to approve or deny it. Sitting back down under a name that
  already has ledger entries counts as a rebuy, so leaving and rejoining can't skip approval.
  Split into its own PR (4b), stacked on this one. The host's own rebuys skip approval. A new request
  replaces the player's last one; leaving the table or ending the session drops it. Approving checks
  again, so a seat that got taken or a rebuy that no longer fits is dropped with a note to the host.
  Requests show to the host above the table, and the player sees they're waiting. e2e: `approval.mjs`.

Notes:
- Both live in the host menu, which is back. End session asks first. Between hands it ends right away.
- Once ended, the server refuses anything that deals, seats or moves chips, and the host menu, open
  seats, start and rebuy disappear. Everyone, including late joiners, gets a settle-up dialog; the
  players panel keeps the ledger after it's closed. A hint shows when the table closes.
- When the hour is up, anyone still there gets error code `closed` and goes back to the join form.
- e2e: `endsession.mjs`.

### PR 5: add stats panel

- [x] Snapshot every player's stack at the end of each hand, bounded (last 1000 hands).
- [x] From the snapshots: chips over time per player, hands played and won, biggest pot won.
- [x] Chart colors come from the design tokens in `web/styles/index.css`.
- [x] Detailed stats (how often each player puts money in, etc.) need hand history. See PR 7.

Notes:
- Snapshots go out in their own `update-stats` message: everything kept on joining, then one per hand
  as it ends. They don't ride on `update-game`.
- A snapshot has the stacks after the hand, who was dealt in, and the pots each winner took (their own
  chips included, matching the "wins N" log). Split pots share evenly; odd chips aren't tracked.
- The stats tab: a line per player of their stack by hand, with a legend, end labels for up to 4
  players, and a crosshair tooltip; then a table of hands, wins, win rate, biggest pot and stack.
  Series colors are the dataviz reference's dark palette, validated against `panel`. Past 8 players,
  lines go gray. e2e: `stats.mjs`.

### PR 6: add table panel

- [x] Players, stats and the host's session controls move out of the chat popover into a "table"
  drawer, top left. Chat keeps chat and log. The header host menu becomes a host-only tab.
- [x] Show the table config (stakes, starting stack, seats, timer, password on/off) in one of the
  panel's tabs, read-only for everyone. Later, maybe editable by the host. A "settings" tab, with the
  table name and invite link too; the header drops "nl texas holdem" and the lock icon, which moved
  there. e2e: `settings.mjs`.

### PR 7: add hand history and detailed stats

- [x] Name hand ranks: "two pair, kings and sevens", in the log and per pot in `update-game`
  (`potHands`). The client no longer mirrors `handName`.
- [x] Record every hand on the server (`history.go`): players with start stacks and cards, actions by
  street, board, pots and nets. Capped at 1000 hands, in memory like the rest.
- [x] Records only leave through `viewFor`: your own cards, plus hands shown at showdown
  (`ShownPlayerNums`, the same rule as `GeneratePlayerView`). Spectators see shown hands only.
- [x] `hand-history` sends each hand as it ends, the last 20 on joining, and older pages for
  `get-hands`.
- [x] A "hands" tab: newest first and open, actions by street, a "hands I played" filter. e2e:
  `history.mjs`.
- [x] Play style stats from totals the server keeps by name (`tally`): VPIP, PFR, 3-bet, AF,
  WTSD, W$SD and bb/100, plus each player's best hand shown. Rows stay gray under 20 hands.
- [ ] Keep the log on the server so late joiners and reloads get it (`log-history`).

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

## 6.4 Drift from the 5.x features

Stage 6 was done before host controls, bots, the ledger and stats, which brought some drift back.

### PR 1: fix seats on phones

- [x] Side seats sat flush against the screen edge: the seat ring was inset by exactly half a plate.
  The inset is now half a seat plus 0.5rem at every breakpoint.
- [x] The dealer button, blinds and bets sat 60% of the way to each seat, which on a narrow table put
  them under a side seat's cards. They now stay far enough in to clear the seat (`markerAt`).
- [x] Bot badges truncated to "mediu…" on phone plates, with the host's remove "×" on top. Small
  plates show just "bot" (the level is on hover and in the players panel), a long name truncates
  before the badge does, and "×" sits on the plate's corner.
- [x] `layout.mjs` checks every whole seat keeps a margin from the screen edges and markers clear every
  seat; `bots.mjs` checks phone badges are whole and "×" doesn't cover them.

Not fixed: a full nine-seat table on a 360px phone still crowds at the top, with one seat's cards over
the next seat's plate. That was already so on main and needs its own layout change.

### PR 2: name the colors

- [x] Semantic tokens for the colors 5.x repeats under raw names: `line`, `field`, `badge`,
  `accent-hover`, `on-accent`, `error`, `warn`, `gain` and `loss`. Each is the shade it replaced, so
  nothing looks different. Card suits, the felt, the table markers and the connection dot keep their
  own colors.

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

- e2e cleanup: move the timing helpers each script reinvents into `e2e/lib.mjs` (act on a control
  once it's enabled, retrying when the turn moves on; wait for a state rather than any update), trim
  checks that pin exact text or pixels, and consider a non-blocking CI job so the suite protects
  main. See the guidance in `e2e/README.md`.
- Pre-action buttons (check/fold, call any) and keyboard shortcuts
- Run it twice, straddles, antes, show/muck at showdown
- Pot-limit Omaha
- Sit-n-go / tournament mode with a blind schedule
- Accounts and persistent stats (only if this should be more than a play-with-friends app)
