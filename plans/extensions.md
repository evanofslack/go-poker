# Game extensions

**Status: planned.** Stages 14-18 in `todo.md`: time bank, a game refactor, Omaha, bomb pots and
double board, short deck, then a backlog of smaller variants. Each engine change gets `holdem/` tests, including `rollout_test.go`'s random
hands, before any table or web work, as in `straddle.md`.

## Decisions
- One engine. `holdem` keeps its name and learns a few orthogonal knobs in `Config.Game`: hole
  cards (2, 4, 5), deck (full, short), limit (no, pot). Any combination the deck can deal is
  allowed, so short deck PLO comes free.
- Bomb pots and double board are per hand, not a game type: `StartHand` takes a `Deal` saying
  what kind of hand this is. A table can also play every hand double board.
- Hand strength lives in one place, `holdem/rank.go`. The client stops evaluating hands: the
  server sends each player the name of their own made hand.
- Deck math is checked up front: `Seats * Hole + 5 * Boards <= deck size` (no burn cards), in
  `Settings.Validate`. Seats can't change after create, so a game that doesn't fit is refused.
- The 7-2 bounty becomes a worst-hand bounty: 7-2 in full deck hold'em, J-6 in short deck, off in
  Omaha (Validate refuses it). The JSON field stays `sevenDeuce` so saved settings load. Stand-up
  works in every game.
- Run it N times is off on double board hands (`MaxRuns` is 0).
- Bots play every game, badly but legally. The medium bot (`bot/medium.go`: Chen preflop score,
  equity by dealing opponents random hands, draws, styles) is built for 2 card hold'em, so each
  stage says what it needs.

## Rules

### Pot-limit Omaha
- 4 hole cards (PLO5: 5). The hand is **exactly 2 hole cards and 3 board cards**.
- The biggest bet or raise is to `top + pot + owe`, where `pot` is every chip committed this hand
  (this street's bets included) and `owe` is what the player has to call. Blinds 1/2, under the gun
  can raise to 7. Min raise is unchanged from no limit.
- A short all-in still doesn't reopen betting.

### Short deck (6+ hold'em)
- 36 cards: the 2s through 5s come out, 6s stay.
- Flush beats full house.
- A-6-7-8-9 is the lowest straight (and straight flush).
- Trips vs straight varies by room. Default Triton: trips beat a straight. A setting
  `ShortTrips` flips it. (Leave out if nobody asks for it.)
- Often played with antes and a button blind and no SB/BB. First version uses the table's normal
  blinds and ante; the button blind comes later (backlog).

### Bomb pots
- Everyone dealt in puts in the bomb amount as dead money, like an ante. No blinds, ante or
  straddles that hand.
- No preflop betting: the flop is dealt and betting starts left of the button, as on any
  postflop street. Min bet is the big blind.
- A stack smaller than the bomb is all in for what they have; side pots as usual.
- The button moves as for any hand.

### Double board
- From the flop on, two boards are dealt, a street at a time each. One round of betting per
  street.
- At showdown each pot splits in half, one half per board, odd chip to board 1; each half to the
  best hand on that board. Same as run it twice, which `showdown()` already does.
- Usually played as a bomb pot, but works for any hand.

### Time bank
- Each seat holds time cards. One card adds `TimeCard` (default 20s) to the turn timer.
- A player sits down with `TimeBank` cards (default 2) and earns one every `TimeEvery` hands dealt
  in (default 20), up to `TimeMax` (default 5). Cards stay with the seat through rebuys and sitting
  out, and are lost on leaving the seat.
- On your turn a "+20s" button spends a card. When the turn timer runs out, a card is spent for a
  connected player who has one instead of checking or folding for them. Disconnected players
  never auto-spend, so they don't stall the table.
- Only with a turn timer on. Bots never use cards.

## Stage 14: time bank

Independent of everything else, small, so it goes first.

### Table (`table/`)
- Settings: `TimeBank uint` (cards to start, 0 off), `TimeCard uint` (ms, 5-60s),
  `TimeEvery uint` (hands, 0 never earn), `TimeMax uint` (cards, 1-10). Validate, log changes in
  `settingsChanges`, editable between hands.
- `Seat.TimeCards int`, `Seat.TimeHands int` (hands since the last card). Set on taking a seat,
  cleared with the seat.
- Command `AddTime{Person}`: their turn, cards left, turn timer armed. Spends a card, moves
  `Timers[TimerTurn]` by `TimeCard`, logs "x added 20s". Errors otherwise.
- `TimerTurn` fired: if the player to act is connected with a card, spend it and re-arm instead of
  acting. Log it as auto.
- `afterHand`: each seat dealt in gets `TimeHands++`; at `TimeEvery` a card, up to `TimeMax`.
- View: `timeCards` per seat (public). `TurnEndsIn` already carries the moved deadline.

### Server
- `add-time` inbound in `messages.go`, mapping in `client.go`, validation in `validate.go`.
- `just types`.

### Web
- `actions.ts`: `addTime()`.
- Footer during your turn: `+20s` button (`default`, with the card count), hidden with no cards.
- Seat: small card count badge near the timer, or only in the seat menu. Decide on screenshots.
- `SettingsFields`: time bank section, shown when the turn timer is on.
- `Countdown` already follows `turnEndsIn`; check it restarts its animation when the deadline
  moves.

### Tests
- `table/timebank_test.go`: add time moves the deadline and spends a card; no cards, not your turn,
  no timer, between hands are errors; timeout auto-spends for connected, acts for disconnected and
  when empty; earning every N hands, capped at max; new seat starts with `TimeBank`; leaving the
  seat clears; rebuy keeps; bots never spend; settings validate and log.
- `server` validate test for `add-time`.
- Restore: a saved table keeps cards (fields are plain JSON, zero is fine for old saves).
- e2e `timebank.mjs`: short turn timer, click +time, see countdown jump and count drop; let it run
  out and see the auto card. Add to `e2e/README.md`.

### PRs
1. time bank table and server
2. time bank web

## Stage 15: game refactor (no visible change)

Make room for more than two hole cards and more than one way to rank hands. Hold'em plays
exactly as now, which the existing tests prove.

### Engine (`holdem/`)
- `Player.Cards []Card` instead of `[2]Card`. JSON is the same array, so saved tables load. Same
  for the `Dealt` and `Shown` events.
- `Config.Game Game`:
  ```go
  type Game struct {
      Hole  int  `json:"hole"`  // hole cards, 0 means 2
      Short bool `json:"short"` // 36 card deck, 6s up
      Pot   bool `json:"pot"`   // pot limit
  }
  ```
  Zero value is no-limit hold'em, so old saves and settings stay as they are. Only
  `Hole` is used in this stage.
- `holdem/rank.go`: `func (g Game) Best(hole, board []Card) (cards []Card, score int)`, lower is
  better. Hold'em: best five of hole plus board (handles 5, 6, 7 cards, also used for records with
  a partial board). Replaces every direct `eval.BestFiveOfSeven` call: `showdown`,
  `table/records.go` `bestHand`, bot.
- `shuffle` takes the `Game` so the deck can change later.
- `view.go` hides cards as zero cards of the same length, so clients still know how many backs to
  draw.

### Table, server, bots
- `records.go`: `bestHand(cards []holdem.Card, …)` through `Game.Best`.
- `sidegames.go`: `isSevenDeuce` takes a slice, false unless 2 cards.
- `bot/`: `spot.hole []eval.Card` and `spot.game holdem.Game` (the view carries `Config`).
  - `bot.go`: `strengthOf`/`best` score through `Game.Best`.
  - `medium.go`: `equity` takes the game: its deck (`holdem.Deck(game)`, exported from the shuffle
    code instead of `eval.DefaultDeck`), opponents dealt `Hole` cards, `score` through `Game.Best`.
    `drawOf`, `connects`, `rankMask` take a slice. `chen`, `suitedPlayable` and `plays` stay 2 card
    hold'em; they get a game switch in later stages.
  - Behaviour must not change for hold'em: `baseline_test.go` and `sim_test.go` numbers hold.
- Server-side hand hint: `table.ViewFor` adds `madeHand []string` for the viewer's own seat, one
  per board (one for now), named from `Game.Best` with `HandName`'s kind-only form. Hidden when
  folded or between hands, as the client does now.

### Web
- Drop `madeHand` from `lib/hands.ts`; `Seat.tsx` reads `view.madeHand`.
- `Seat.tsx`, `Card.tsx`: lay out `player.cards.length` cards, not two. Overlap them past two so
  4 and 5 fit, own cards included (`bigCardSize`), phone and `short:` too. `MiniCards` already
  takes a count.
- Hand history shows any number of hole cards.
- `just types`.

### Tests
- `holdem/rank_test.go`: hold'em `Best` equals `eval.BestFiveOfSeven` on many random hands; 5
  and 6 card inputs.
- Existing `holdem`, `table`, `bot` tests pass unchanged except for the type, including
  `bot/baseline_test.go`.
- `table` view test: hidden cards keep their count; `madeHand` only for the viewer.
- e2e: full run, screenshots of seats on desktop and phone (`SHOTS=`) to check nothing moved.

### PRs
3. hole cards as a slice, `Game.Best` (engine, table, bot, server)
4. server hand hint, cards by count (web)

## Stage 16: pot-limit Omaha

### Engine
- `StartHand` deals `Hole` cards (default 2).
- `Game.Best` for `Hole > 2`: every 2 of the hole cards with every 3 of the board, through
  `eval.HandValue` (60 for PLO, 100 for PLO5). Works with a 3 or 4 card board for records and hints.
- `Legal`: with `Pot`, `RaiseMax = min(p.Bet + p.Stack, top + committed + owe)`, `RaiseMin`
  capped by it as now. `committed` is the sum of every player's `Committed`.
- `MaxRuns` deck math already uses `len(Deck)`, so it adjusts by itself.

### Table
- Settings `Game string`: `"holdem"` (default), `"omaha"`, `"omaha5"` mapped to `Hole`; `Limit
  string`: `"no"` (default), `"pot"`. Pot limit allowed with hold'em too. Changeable between hands,
  logged ("the game to PLO").
- Validate: deck math (9 seats × 5 + 5 = 50 fits; short deck comes later), 7-2 only with 2 cards.
- Records: `HandRecord.Game` label ("NLH", "PLO", "PLO5") so mixed sessions read right.
- Bots: an Omaha preflop score in `medium.go` next to `chen` (pairs, double suited, connected, high
  cards; good enough), and `plays` (which opponent hands the equity sim keeps) using it. Easy bot's
  `preflop` gets the same switch. Postflop equity already goes through `Game.Best`; trials may need
  trimming for PLO5 speed. The table clamps a bot raise to `Legal.RaiseMax` so a bot
  can't make an illegal pot-limit bet.

### Web
- `SettingsFields` / `CreateTable` / `EditSettings`: game `Segmented` (Hold'em, Omaha, Omaha 5)
  and limit (No limit, Pot limit).
- Header game info: "PLO 10/20".
- `RaiseInput`: with pot limit the presets are ½ pot, ¾ pot and Pot (= `raiseMax`, server's
  number). Fractions: `top + f * (committed + owe)`, clamped to min/max.

### Tests
- `rank_test.go` Omaha: a four-flush board with one suited hole card isn't a flush; a board straight
  doesn't play; quads on board play as two pair (or trips) at best; wheel with two hole cards; PLO5
  picks the right two.
- `betting_test.go` (or `potlimit_test.go`): blinds 1/2 UTG max 7; after UTG pots to 7, the next
  player re-pots to 24 (top 7 + 10 committed + 7 owed); with antes and straddles; short
  stack capped by stack; short all-in doesn't reopen; no limit unchanged; min raise above pot is
  clamped to pot.
- `rollout_test.go` runs random hands for PLO and PLO5: chips conserved, no duplicate cards, every
  `Raise` within `Legal`.
- `table`: settings validate and log, 7-2 refused, game change between hands, records label.
- `bot/sim_test.go` across games; medium beats easy at PLO; `equity` deals 4 or 5 cards and no
  duplicates.
- `table/fuzz_test.go` across games.
- e2e `variants.mjs`: create a PLO table, see 4 cards each, Pot button stops at the pot.
  Screenshots of 4 and 5 card seats on phone.

### PRs
5. omaha engine (deal, rank, pot limit)
6. omaha table, bots and web

## Stage 17: bomb pots and double board

### Engine
- `StartHand(d Deal)` instead of `StartHand(straddles int)`:
  ```go
  type Deal struct {
      Straddles int  `json:"straddles"`
      Bomb      uint `json:"bomb"`   // dead money from everyone, straight to the flop; 0 for none
      Boards    int  `json:"boards"` // 0 or 1 for one, 2 for double board
  }
  ```
- Bomb: no blinds, ante or straddles (straddles with a bomb is `ErrIllegal`). Everyone dealt in
  `putDead(Bomb)` as `Acted{Kind: KindBomb}`. Then straight to the flop as `NextStreet` would deal
  it: `MinRaise` the big blind, `ToAct = Button`, `advance()`. If fewer than two can bet, cards run
  out as for an all in.
- `State.Bomb uint` for the view and records.
- Live boards: `Boards [][]Card` holds every board being played, from the flop, instead of only
  finished runouts. `Board` stays as `Boards[0]` so the client and records keep working.
  - `NextStreet` deals the street to each board in order, one `StreetDealt` per board (`Run` is
    the board index; consider renaming to `Board` while the client is being touched anyway).
  - On single board hands run it N times keeps working by appending boards after `RunFrom` at the river, as now.
  - `showdown()` already splits each pot across `Boards`.
  - `MaxRuns` is 0 when `Boards > 1`. `Hunt` runs out every board.
- Deck math for double board: `Hole * players + 10`.

### Table
- Settings: `BombPot uint` (amount in chips, 1-20 bb), `BombEvery uint` (every N hands, 0 for only
  when the host asks), `BombDouble bool` (bomb pots are double board), `DoubleBoard bool` (every
  hand double board). Validate deck math with 2 boards when either is on.
- Host action `bomb-pot`: the next hand is a bomb pot. `State.BombNext bool`, shown to everyone,
  host can cancel. `BombEvery` sets it in `afterHand` on the Nth hand. No straddle ask before a bomb.
- `scheduleDeal`/deal builds the `Deal` from settings and `BombNext`.
- Records: `HandRecord.Bomb uint`; `Boards` already recorded; `HandPlayer.Hand` per board when
  there are two (`Hands []string`).
- Stats: bomb pots don't count for VPIP and PFR (no preflop decision); they count for hands and
  saw flop.
- Side games: 7-2 counts a win on either board (like runs); stand-up the same.

### Web
- `CommunityCards`: two boards stacked during play, reusing the run-it layout; winning cards per
  board at showdown. Phone and `short:` layouts need screenshots.
- Hand hint shows one made hand per board.
- Host menu: "Bomb pot next hand" (and cancel). Table shows a "Bomb pot" badge before and during
  the hand.
- `SettingsFields`: bomb pot section (amount, every N hands, double board) and double board toggle.
- History: bomb label, both boards.

### Tests
- `holdem/bomb_test.go`: everyone posts, no blinds, starts on the flop with first to act left of
  the button (and heads up, the big blind side); short stack all in for less makes a side pot;
  everyone all in from the bomb runs out to showdown; straddles with bomb illegal; button moves.
- `holdem/boards_test.go`: two boards dealt each street from the same deck, no duplicates; pot
  split half each with odd chip to board 1; scoop; split on one board; side pots across boards;
  fold win takes everything before the flop and after; run it refused; hunt fills both boards.
- `rollout_test.go`: random bomb and double board hands for every game.
- `table`: host bomb next hand and cancel; every N hands; bomb skips the straddle ask; settings
  validate; records and stats; 7-2 on board 2.
- `server` validate for the host action.
- e2e `bombpot.mjs`: host bombs a hand, everyone posts, flop out with no preflop action, double
  board shows two rows, pot split.

### PRs
7. bomb pot engine
8. live boards and double board engine
9. bomb pots and double board table and server
10. bomb pots and double board web

## Stage 18: short deck

### Engine
- Deck: `Short` filters 2-5 out of `eval.DefaultDeck`: 36 cards.
- `Game.Best` with `Short`: try each five (any five for hold'em, 2+3 for Omaha), score with
  `eval.HandValue`, then remap into a short deck order that's still one int, lower better:
  - flushes (Cactus Kev 323-1599) swap ahead of full houses (167-322), each keeping its order;
  - A-6-7-8-9 scores as the unused nine-high straight slot (1605), suited as the nine-high straight
    flush slot (6);
  - Triton: trips (1610-2467) swap ahead of straights (1600-1609).
- `HandName` takes the `Game` (or names from the remapped score), so names still read right,
  including "a straight, nine high" for A-6-7-8-9.
- Stats `BestScore` compares remapped scores; fine within a game.

### Table
- Settings `Deck string`: `"full"` (default), `"short"`. Validate deck math against 36: hold'em 9
  seats (23), PLO up to 7 seats (33), PLO5 up to 6 (35), double board less.
- Records label "6+".
- Worst-hand bounty: `isSevenDeuce` becomes `isWorstHand(game, cards)`: 7-2 on a full deck, J-6
  (any suits) on a short deck, false in Omaha. Side game kind `"J-6"` for logs, records and the
  show claim; the web label follows the game ("7-2 bounty" / "J-6 bounty").
- Bots: the equity sim already uses the game's deck. `chen` shifts on a short deck (fewer, stronger
  hands; small cards count less); a small adjustment is enough.

### Web
- `SettingsFields`: deck `Segmented`, with a hint for the rule changes.
- Header: "6+ hold'em".

### Tests
- `rank_test.go` short: flush beats full house; A-6-7-8-9 beats nothing above it and loses to
  6-7-8-9-T; suited A-6-7-8-9 straight flush; trips beat straight (and the other way with
  `ShortTrips`); names; short deck PLO uses exactly two.
- Deck: 36 cards, no 2-5, same seed same deck.
- `table/sidegames_test.go`: J-6 pays on short deck and 7-2 doesn't; 7-2 still pays on a full
  deck; nothing pays in Omaha; switching deck mid-session pays by the hand's game.
- `bot`: short deck sim plays legal hands; equity never deals a card below 6.
- `rollout_test.go` short deck, hold'em and PLO.
- `table`: deck math refuses too many seats for the game, logs.
- e2e: `variants.mjs` creates a short deck table, no card below 6.

### PRs
11. short deck engine
12. short deck table and web

## Backlog

Smaller, after the above. Each a few hours to a day.
- **Big blind ante**: the big blind posts everyone's ante as dead money, blind first so a short big
  blind keeps their blind. `Config.BBAnte bool`.
- **Button blind**: the short deck format. Everyone antes, the button posts a blind, no SB/BB, first
  to act is left of the button. `Positions`/`StartHand` need blinds to be optional.
- **Dealer's choice / mixed games**: the game rotates each orbit (`Rotation []Game`) or the button
  picks it. Settings already apply between hands through `s.Next`.
- **Hi-lo (PLO8, Big O)**: a low evaluator (eight or better), pots split high/low and quartered.
  Changes `showdown()` and the pot result shape. Bigger than the rest.
- **Crazy pineapple**: 3 hole cards, discard one after the flop. A new `Discard` action and step.
- **Tournament mode**: blind levels on a timer, no rebuys, payouts at the end.
- **More side games**: anything that pays between stacks fits `sidegames.go`.

## Order

| Stage | PRs | Depends on |
| --- | --- | --- |
| 14 time bank | 1-2 | nothing |
| 15 refactor | 3-4 | nothing |
| 16 Omaha | 5-6 | 15 |
| 17 bomb / double board | 7-10 | 15 (16 for PLO bombs) |
| 18 short deck | 11-12 | 15 |

Built as stacks of PRs, one worktree per stage, landed with `gh stack merge`: 14 and 15 in
parallel, then 16, 17 and 18 each from main after the last lands. Each stack's top PR checks its
stage off in `todo.md`.

Run `just check` for each PR; `just e2e` for the touched area as you go and the whole suite before
each web PR, with screenshots for seats, boards and settings.
