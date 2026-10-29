# Hard bot

**Status: done.** A third bot level that reads ranges, not just pots, and plays every game the
table deals. Also fixes medium's leak to big bets. Replaces "Later: hard bot" in `todo.md` 5.2.

## Where medium stands

Medium (`bot/medium.go`) compares equity to pot odds. Its equity deals opponents hands from one
range for everyone: Chen 3+ (6+ in a raised pot), and against a bet this street, hands that connect
with the board or 15% bluffs. So:
- A bet's size doesn't change what medium thinks the bettor holds. A big bet only makes the price
  worse, so it folds more to overbets, and calls small bets too light.
- Checks don't cap a range, earlier streets are forgotten, and every opponent is the same player.
- Bluff rates are fixed numbers, not tied to the size it bets.
- No stack depth: no push/fold when short, no committing at a low stack-to-pot ratio.
- Bomb pots still treat opponents as Chen 3+ though nobody chose to play. Double board equity only
  looks at the first board.

## Decisions

- Same shape as medium: hand-written, Monte Carlo equity on the engine's `Game.Best`, no solver,
  no training. Pluribus-style CFR doesn't fit a home-game server; ranges and sizes get most of the
  way there.
- **Ranges by weight.** Each opponent's hands are weighted by how likely their actions this hand
  were with that hand: their preflop line by position, then each bet, call, check or raise after
  the flop, by the hand it made on that street's board. Bigger bets weight strong hands and air
  over middling ones (polar). The equity sim deals opponents hands by these weights.
- **Preflop by percentile.** Each starting hand gets a percentile in its game: for two cards from
  equity tables worked out once per deck (full and short) at first use, for Omaha from the Omaha
  score over sampled hands. Ranges are "top X%" by position, so one chart covers every game.
  Push/fold below 15 big blinds in no limit hold'em.
- **Decisions by EV.** Call when equity against the weighted range beats the price. Raise or bet
  for value when equity against the hands that would call is ahead. Bluff with hands that can't
  win at showdown, draws and blockers first, at about `bet / (pot + 2 bet)` of the betting range,
  less multiway.
- **Sizes.** A third pot on dry boards, two thirds to most of it on wet ones, overbets on the river
  in no limit with the nuts and some bluffs. All in when the stack-to-pot ratio is low.
- **Reads.** The bot gets each opponent's public totals (VPIP, PFR, 3-bets, aggression, hands) and
  their actions this hand. Totals shift the preflop ranges, shrunk toward a default until there
  are enough hands. Nothing private: totals are public and actions are seen by everyone.
- **Variants.** Pot limit caps sizes (already in `size`). Omaha and short deck come from the
  game's deck, `Best` and percentiles. Bomb pots start opponents at any hand. Double board equity
  is half on each board.
- **Medium's overbet fix**: medium's bettor range narrows with the bet's size too (a small change
  in `plays`), so the fix reaches medium without the rest of hard.
- Hard is opt-in. The add-bot dialog keeps medium as the default.

## Benchmarks

`bot/sim_test.go`, run with `BOTSIM=1`.
- **Duplicate matches**: each heads-up match is played twice from the same seed with seats
  swapped, stacks reset to 100 BB each hand, so the cards cancel out. Report bb/100 with a 95%
  interval over hands.
- **Exploiters** (test only, `exploit_test.go`): a station that never folds a pair, an overbettor
  that jams turns and rivers with anything, a nit that plays the top 10% and only reraises QQ+, and
  a trapper that check-raises its strong hands.
- Hard must beat medium and every exploiter, heads up and at six handed, in NLH, PLO and short
  deck, with bomb pots and double board on, and medium must stop losing to the overbettor.
- A decision should take a few milliseconds: it runs in the table's actor.

## PRs

Stacked:
1. **bench and overbets**: bots get the hand's actions and reads, duplicate matches, exploiters;
   medium's bettor range by size, bomb pots and double board.
2. **hard**: `bot/hard.go`, percentiles, weighted ranges, sizes, push/fold, sims.
3. **table and web**: `hard` level valid on the server, add-bot picker, home page bots tile,
   `e2e/bots.mjs`, todo.

## Results

`BOTSIM=1 go test ./bot -run TestSimHard -v`, 10 shards of 500 hands, each played twice, in bb/100
a seat with a 95% interval, at the end of the work:

| hard against | heads up | six handed |
| --- | --- | --- |
| medium, NLH | +32.9 ± 13.5 | +14.1 ± 5.3 |
| medium, PLO | +203.6 ± 37.0 | +123.7 ± 19.8 |
| medium, short deck | +40.0 ± 23.0 | +33.2 ± 11.3 |
| medium, double board bomb pots | +10.4 ± 11.9 | +37.3 ± 11.8 |
| station | +173.3 ± 31.1 | +310.9 ± 30.4 |
| jammer | +31.6 ± 24.7 | +12.8 ± 9.4 |
| nit | +48.6 ± 10.6 | +8.7 ± 6.2 |
| trapper | +32.3 ± 29.3 | +15.7 ± 9.3 |

Six handed it plays about 25% VPIP, 16% PFR, 7% 3-bet, a solid regular's numbers.

Differences from the plan:
- The jammer showed polar overbets beat medium because they're good, not because medium folds
  too much: after the fix medium's calls against them break even. Hard overbets the river itself.
- The fold model was the hard part. Hand strength against random hands overrated folds; it now
  defends each opponent's own range as often as they've folded to bets, which needed a new
  total, fold-to-bet, and leaving bomb pots out of preflop stats.
- Double board strength is the percentile of a hand's average over the two boards.
- Two card charts are generated (`go test ./bot -run TestChart -update`) and checked in, since
  working them out takes half a second.
