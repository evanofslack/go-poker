# Feature checklist

Everything the server does today, written down before the engine/table rewrite (see
`refactor.md`) so nothing is dropped. The cutover PR ticks these off. Web-only features (layout,
stats charts, invite links) aren't listed; they keep working as long as the data below reaches
the client.

All ticked at the cutover: covered by the `table` tests, the `server` tests and the e2e scripts.

## Lobby and joining
- [x] Create a table with a name (≤32 chars), username (≤20, not "system") and optional settings;
      missing settings take defaults (8 seats, 10/20, buy-in 1000–4000, 30s turns, 5s between hands).
- [x] Settings validated: 2–9 seats, 1 ≤ sb ≤ bb ≤ 1,000,000, bb ≤ min buy ≤ max buy ≤ 1000 bb,
      turn time off or 10–300s, next hand delay 2–15s. Fixed after creation.
- [x] Optional table password (≤32 chars); joining needs it, with codes `password-required` and
      `wrong-password`. The creator, and anyone who joins with it, gets it back for invite links.
- [x] Join by name; names are unique per table, case-insensitively, including seated players who
      are disconnected.
- [x] Creator becomes host; if a table has no host the next joiner does.
- [x] Public tables listed on the home page (`list-tables`, works before joining): name, blinds,
      buy-ins, seats, players, bots, watchers, running, paused, has password. Ended tables
      hidden. Sorted: not full first, then most players, then name.
- [x] Limits: max tables (100), tables per creator IP (3), connections total (1000) and per IP
      (10), messages per second (20) per connection. All configurable from env.
- [x] Joining sends: joined confirmation (table, name, id, password), the game view, all kept hand
      stats and totals, the latest page of hand history, the log backlog.
- [x] Chat: messages ≤200 chars, sent under the server-known name; "X has joined" and
      "X rejoined, continuing their ledger" system messages.
- [x] Leave the table (`leave-table`): gives up the seat, passes host if needed.

## Seats and stacks
- [x] Take a seat 1..seats with a buy-in in [min, max]; seat must be free; can't hold two seats.
- [x] Taking a seat gives a reclaim token, sent only to that player.
- [x] A seated player is dealt in from the next hand.
- [x] Leave seat: between hands right away; mid-hand folded when action reaches them and removed
      after the hand. Cashes out the stack to the ledger. Client becomes a watcher immediately.
- [x] Sit out / sit back in (toggle). Mid-hand it applies after the hand ("will sit out the next
      hand"), and toggling again cancels. Busted players can't sit back in without chips.
- [x] Busted players are sat out automatically.
- [x] Rebuy between hands: busted needs ≥ min buy; topping up any amount; stack can't pass max
      buy. Deals a busted player back in.
- [x] Rebuy approval setting: rebuys, and a returning player sitting back down, wait for the
      host (except the host's own). Requests listed oldest first; a new request replaces the
      player's old one. Approve re-checks and drops a request that no longer fits; a mid-hand
      rebuy approval waits for the hand to end. Deny. Requests cleared when the requester leaves
      and when the session ends.

## Hand flow
- [x] Any seated player, or the host without a seat (bots-only game), can start the game; needs
      2+ players ready. Not while paused.
- [x] Once running, the next hand is dealt `nextHandDelay` after a hand ends, when 2+ are ready;
      otherwise "waiting for players" and deals once someone sits/rebuys/returns.
- [x] Button moves to the next ready seat each hand; heads-up the button posts the small blind.
- [x] Blinds posted (all-in if short). Min raise = big blind; a raise must be at least the last
      raise; short all-ins allowed and don't reopen betting for players who already acted.
- [x] Check, call (all-in if short), bet/raise, fold. Only on your turn ("not your turn").
- [x] After a street's betting closes, pause `streetDelay` (1s) showing the last action, then
      deal the next street.
- [x] When no more betting is possible, uncalled chips go back and the board runs out; all-in
      players' cards are shown.
- [x] Showdown: best five of seven, side pots per all-in level, split pots, odd chips to the
      first winner left of the button. Hands still in at showdown are shown. Fold-win shows no
      cards.
- [x] Between hands the last result stays visible: winners, pot amounts, winning hand names
      ("two pair, kings and sevens") and cards.
- [x] Each seat shows its action this street ("check", "raise to 80", "all in"); between hands
      the last street's stay up.
- [x] Turn timer (optional): deadline sent to clients; on timeout check if possible, else fold;
      history marks it as a timeout.

## Reconnects and leaving
- [x] A seated player who disconnects keeps their seat (and host role) for `reconnectGrace`
      (60s); "X disconnected" logged; listed as disconnected. They still get folded/checked by
      the turn timer.
- [x] Rejoin with the token reclaims the seat and name without the password, even from a new
      tab; a stale connection holding the seat is closed. "X reconnected" logged.
- [x] Grace expiry removes them like leaving the seat, and passes host if it was theirs.
- [x] A watcher (or seated player leaving by choice) leaving drops their buy request.
- [x] Host leaving passes host to a connected seated player, else the first connected watcher by
      name, else nobody.
- [x] Table deleted when nobody is connected and nobody is in grace (unless the session ended).

## Host controls
- [x] Make host (must be connected, not yourself).
- [x] Set banker (connected or in the ledger); defaults to the host.
- [x] Move a player to watching (unseat, cash out).
- [x] Remove a player (unseat if seated, kick their connections with code `removed`). Not
      yourself. No bans.
- [x] Sit a player out (now between hands, else after the hand); only they can sit back in.
- [x] Adjust chips: add or remove (≤1e9, optional reason ≤60 chars) between hands for that
      player; adding can't pass max buy and deals a busted player back in; removing to 0 sits
      them out. Ledger records host-add / host-remove with who and why.
- [x] Pause / resume: no new hands while paused; mid-hand the hand plays out first.
- [x] End session: now between hands, else after the hand. Stops dealing, clears requests,
      ledger final; table closes after `endedTableTTL` (1h) with code `closed`. After ending,
      seat/start/rebuy/sit-out/approve/deny/adjust/pause/end/bot actions are refused.
- [x] Add bot (host only): seat, buy-in, name (unique), level (easy, medium). Remove bot by
      seat (folded on turn mid-hand, removed after).

## Bots
- [x] Bots act after a random 1–2s delay, from their own player view only.
- [x] Invalid bot moves fall back to check, then fold.
- [x] Busted bots rebuy their original buy-in between hands (logged, in the ledger).
- [x] Bot levels shown to clients.

## Records
- [x] Ledger by name: buy-in, rebuy, cash-out, host-add, host-remove entries with time, who, note.
      Rows in order of first buy-in: bought in, cashed out, stack (counting bets this hand), net;
      banker; balance (sum of nets).
- [x] Hand history: last 1000 hands as played (hand number, start time, blinds, dealer, players
      with seat/starting stack/net, actions by street with amounts/to/all-in/timeout, board,
      pots with winners and hand names). Own cards visible; others only if shown. Paged 20 at a
      time with `get-hands before=N`; new hands pushed as they finish.
- [x] Hand stats: per hand snapshot (number, start, stacks after, names dealt in, chips won),
      last 1000; pushed per hand, all on join.
- [x] Player totals by name over every hand: hands, VPIP, PFR, 3-bet spots and 3-bets,
      aggressive actions, calls, saw flop, showdowns and wins, net, best hand.
- [x] Table log, last 200 lines, backlog on join: sits, leaves, sit out/back, rebuys, host
      actions, pauses, session end, disconnects/reconnects, "waiting for players".
- [x] Errors go only to the client that caused them.

## Operations
- [x] Idle tables (no player message or join for 30 min, bots don't count) close with a message.
- [x] Shutdown tells every table "the server is restarting" and drains connections.
- [x] Prometheus metrics: hands dealt, limit hits, slow client drops, tables closed by reason,
      grace outcomes, panics; gauges for connections, tables, players (human/bot), limits.
- [x] Slow clients dropped instead of blocking the table; handler panics recovered.
- [x] Origin check, client IP from X-Forwarded-For only behind a trusted proxy.
