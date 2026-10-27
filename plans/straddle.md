# Straddles and run it N times

**Status: planned.** Stage 11 in `todo.md`, after the e2e cleanup (`e2e.md`).

Two house rules that change the betting engine (`holdem/`), unlike stage 9's. Each gets engine
tests, including `rollout_test.go`'s random hands, before any table or web work.

## Decisions
- Straddles are decided before the cards are dealt, one player at a time: under the gun first,
  then each next player can re-straddle at double. Everyone sees each straddle as it goes in.
- The button never straddles, so a table gets at most (players dealt in - 3) straddles: none heads
  up or 3 handed, one 4 handed, two 5 handed.
- The host picks `off`, `optional` or `forced`. Forced posts the first straddle for under the gun;
  re-straddles stay optional.
- You can only straddle for the full amount. No all-in straddles.
- Run it N times only when the hand is all in before the river, only if the table allows it. Only
  players still in the hand vote. Each picks a number of runs; the table runs the smallest pick,
  so everyone agreed to at least that many.
- N goes as high as the deck allows. There are no burn cards, so that's
  `len(Deck) / (5 - len(Board))`: 9 heads up all in preflop, 6 with nine players preflop.
- 7-2 and stand-up count a win on any run. The 7-2 bounty is paid in full once, not per run.

## Straddles

| Dealt in | Most straddles | Biggest |
| --- | --- | --- |
| 2-3 | 0 | |
| 4 | 1 | 2 bb |
| 5 | 2 | 4 bb |
| 6 | 3 | 8 bb |
| n | n - 3 | |

### Engine (`holdem/`)
- `Positions()` returns what the next hand would be without changing anything: `Button`, `SB`,
  `BB` and `Straddle []int`, the seats that could straddle in order, from under the gun up to the
  seat before the button, cut off at the first who can't cover their straddle (after the ante).
  `StartHand` uses the same code to place the button and blinds, so the two can't disagree.
- `StartHand(straddles int)` deals as now, then posts `Positions().Straddle[:straddles]` after the
  blinds: the k-th straddle (from 0) is `BigBlind << (k+1)`, put in with `put` like the blinds, as an
  `Acted` with a new `KindStraddle`. More straddles than `Positions` allows is `ErrIllegal`.
- `MinRaise` is the last straddle, so after 2 bb the smallest raise is to 4 bb, after 4 bb to 8 bb.
- `ToAct` starts at the last straddler instead of the big blind, so `advance` moves to the next
  player. The last straddler gets the option the way the big blind does now: they haven't `Acted`,
  so `needsToAct` gives them a turn when the action comes back unraised.
- `State.Straddles []int` (and on `View`) for who straddled this hand, cleared by `StartHand`.
- Antes go in first, then blinds, then straddles.

### Table (`table/`)
- Setting `Straddle string`: `"off"` (default), `"optional"`, `"forced"`. Validated in
  `Settings.Validate`, editable between hands like the other house rules, logged when changed.
- Asking: `State.StraddleAsk *StraddleAsk{Seat, Amount, Yes []int}` plus `TimerStraddle` (5s).
  The player answers with a `Straddle{Person, Yes bool}` command. Yes appends to `Yes`, logs "alice
  straddles 40" and asks the next seat in `Positions().Straddle`; no, the timer, or running out of
  seats ends the asking. The hand then deals with `StartHand(len(Yes))`.
- When: the first ask goes out as the pause between hands starts (`afterHand`, alongside
  `scheduleDeal`), with the pause as its deadline, so a table where nobody straddles plays at the
  same speed. Each yes gives the next player 5s. If `TimerDeal` fires while someone is still being
  asked, it waits for the answer. On the first hand the ask goes out when the host presses start.
- Who's skipped: bots, disconnected players and players sitting out (or sitting out next hand) end
  the chain at their seat, as a no. Bots still post a forced straddle.
- Forced: under the gun's straddle is in `Yes` from the start, without asking, if they can cover it;
  the ask starts at the next seat.
- Seats changing: after any command between hands, if `Positions()` no longer starts with the seats
  in `Yes` plus the one being asked, the asking starts over from under the gun. Pausing, ending the
  session, or turning straddles off clears it.
- Inbound action `straddle` with a `yes` field: `server/client.go`, `server/validate.go`, a sender
  in `web/actions/actions.ts`. `StraddleAsk` goes on the view (everyone sees who's deciding).
- Records: history reads "posts straddle 40". `records.go` treats `straddle` like the blinds when
  it lists actions, and `tally` (`stats.go`) skips it, so it isn't VPIP, and it isn't a raise for
  PFR or 3-bets.
- Bots: the medium bot's "nobody raised" is `s.top <= s.bb` (`bot/medium.go`), so a straddle
  reads as a raise. `spot.bb` becomes the biggest blind or straddle this hand (from
  `View.Straddles`). Check bots facing straddles in `bot/sim_test.go`.

### Web
- Footer prompt for the player being asked: "straddle 40?" yes / no with the countdown. Everyone
  else sees "alice is deciding to straddle".
- A toast and log line for each straddle; seats show "STR" next to SB/BB (`Seat.tsx` `Badges`,
  `blind()`).
- The setting in the create form's house rules and the settings tab, as a three-way choice.

### Tests
- Engine: `Positions` at 2, 3, 4, 5 and 9 handed; action order and min raise with one, two and three
  straddles; the last straddler's option; everyone folding to the straddler; a straddler who can't
  cover it is cut off; antes plus straddles; straddles in `rollout_test.go`.
- Table: the chain (yes, no, timeout, re-straddle, stopping at the button), forced, skipping bots
  and disconnected players, seats changing mid-ask, pausing mid-ask, saving and restoring mid-ask,
  stats and records; straddles in the table fuzz test.
- e2e (`straddle.mjs`): four players, under the gun straddles, the next can't (button), the marker,
  preflop action order and the straddler's option.

## Run it N times

### Engine (`holdem/`)
- `State.Runs` (1 by default) and `RunFrom`, the board cards before the extra runs. `MaxRuns()`
  is how many runs the deck allows now. `SetRuns(n)` is legal only mid-hand, not betting, with
  fewer than two players who can bet, before the river, `n` from 1 to `MaxRuns()`, and only once a
  hand.
- `NextStreet` deals the first run street by street as now. Once it reaches the river, with more
  runs to go, each further call deals a whole run (`Board[:RunFrom]` plus fresh cards) into
  `Boards [][]Card` instead of showing down. `StreetDealt` gets a `Run` number. After the last run,
  the showdown.
- Showdown per run: each pot splits into n shares, odd chips to the earlier runs. Each share is
  scored on its own board and paid like a pot now (odd chips left of the button).
  `Pot.Runs []PotRun{Amount, Winners, Cards, Hand, Score}` holds each run; `Pot.Winners` is everyone
  who won any run, so code that asks "did they win this pot" (7-2, stand-up) works unchanged.
  `PotWon` gets a `Run` number, one per pot per run.
- `View` gets `Boards` and `Runs`.

### Table (`table/`)
- Setting `RunIt bool`, "allow running it more than once", default off.
- When the hand is all in before the river (in `played`, where it schedules `TimerStreet` with no
  betting), the table asks instead: `State.RunVote *RunVote{Seats, Picks map[int]int, Max}` plus
  `TimerRunVote` (5s). `Seats` are the players still in. They answer with
  `RunIt{Person, Runs int}`. Once everyone has answered, or the timer runs out, the table calls
  `SetRuns(smallest pick)` and deals on. Unanswered and disconnected count as 1. Bots pick the max,
  so the humans decide.
- Dealing: the first run street by street as now; each extra run all at once, one `StreetDelay`
  apart.
- Inbound action `run-it` with a `runs` field. `RunVote` goes on the view.
- Records: `HandRecord.Boards`; pots per run in history ("run 2: bob wins 200 with a flush");
  `HandStats.Won` adds up each run's share (`records.go` splits by run, not by `Winners`); stats
  count a showdown win on any run. Logs "running it 3 times" or "running it once".
- 7-2: unchanged rule, the 7-2 must be in `Pots[0].Winners`, which with runs means it won the main
  pot on any run. Stand-up: unchanged, any pot's `Winners`.
- Duck hunt is unaffected: run it only happens on hands that go to showdown.

### Web
- Prompt for players still in: "run it how many times?" with buttons 1 to the max and the
  countdown, and who has answered. A no-run-it table never shows it.
- Each extra board under the first (`CommunityCards.tsx`), compact on phones; winners and hand
  names per run; every board in the hands tab.

### Tests
- Engine: pots split per run; the same winner every run; different winners; side pots; odd chips
  per run; `MaxRuns` preflop, on the flop, on the turn; legality (betting, after the river, twice);
  several runs in `rollout_test.go`, with chips conserved.
- Table: the vote (everyone picks 2, one picks 1, smallest wins, timeout, a voter disconnecting, a
  folded player can't vote), the pause before the run out, records and stats, 7-2 and stand-up
  with runs, saving and restoring mid-vote.
- e2e (`runit.mjs`): two players all in preflop, both pick 2, two boards and split pots; one picks
  1, one board.

## PRs
1. **straddle engine**: `Positions`, `StartHand(straddles)`, `KindStraddle`, min raise and order,
   engine tests.
2. **straddle table and web**: the setting, asking, records and stats, the prompt and marker, e2e.
3. **run it engine**: `Runs`, `Boards`, per-run showdown, engine tests.
4. **run it table**: the setting, the vote, records, side games.
5. **run it web**: the prompt, extra boards, history, e2e.

Straddles and run it don't touch each other, so 3 can be written alongside 1 or 2.

## Open questions
- Should a player be able to pre-set "always" / "never" in the seat menu, to skip the prompt?
- Is 5s right for the straddle prompt and the vote?
- Cap the run-it picker (say at 4) even when the deck allows more?
