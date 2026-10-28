# Table polish

**Status: done.** Stage 13 in `todo.md`.

Give the table some depth and short, quiet motion, and a display tab where each viewer picks how it
looks. Before this, every change on the table appeared in the next frame, and everything was
flat: plain green felt, solid card backs, bets as gold pills, and a dealer button that jumped from
seat to seat.

## Decisions
- Tasteful over realistic: simple chips and card backs, light shadows, no tilted 3D table (it
  hurts reading and the seat maths in `lib/seats.ts`).
- CSS only, no animation library and no server changes. Motion is 150 to 500ms and never holds back
  state, since bots act fast and updates can overlap.
- No chips moving into or out of the pot: the bet is gone from the view by the time it would move,
  so it needs an event-driven layer that isn't worth it yet.
- Motion goes through the `motion:` variant: off for `prefers-reduced-motion`, and off until the
  table has settled 500ms after a join or rejoin (`data-motion` on the table), so a reload doesn't
  replay the deal. e2e asks for reduced motion, so checks see where things end up.
- Viewer preferences are client-only, saved in localStorage. Each one is a data attribute on
  `<html>` that overrides tokens in `styles/index.css`, so components don't change.
- No chip is white, so none reads as the dealer button.

## PRs
Stacked. Each runs `just check` and the e2e scripts for its area, with screenshots at phone size.

1. **table depth and chips** (#153): lit felt in a teal rail, flat card backs in a light frame,
   bets and the pot as small chip stacks (color and height by size in big blinds), raised dealer
   and blind buttons, a gold winner glow, and the board lifted when it runs twice.
2. **dealing and flipping** (#154): cards slide in as they're dealt, the flop one after another;
   hidden cards flip over at showdown (the same element, so it turns rather than swaps); folded
   cards slide away.
3. **gliding dealer button** (#156): one dealer button, drawn by the table rather than by each seat, so
   it glides to the next seat each hand. Each seat's markers run toward the middle, with the
   button just outside them. Blinds only show preflop, so they pop in rather than glide. Phones
   keep both on the seat plates.
4. **stack changes and turn pulse** (#158, landed in #162): stacks and the pot count to their new value; the seat to act
   pulses once as its turn starts.
5. **options tab** (#159, landed in #162): a `prefs` object in the store (one localStorage key,
   merged with defaults, one `setPref` action), with the keyboard shortcuts toggle and the mute
   moved into it. Looks are data attributes on `<html>`, set before first paint by a small script
   in `_document.tsx`.
   - [x] animations: system, on or off (#159)
   - [x] deck: four color or two (#163)
   - [x] table and card back colors, ten each, picked from swatches (#163)
   - [x] recorded sounds (Kenney, CC0) by group: turn (always, when away, off), turn timer,
     decisions, cards, chips and chat, with a volume, the header's full mute, and a test button
     each; a hidden tab keeps only your own (`lib/sound.ts`, #165)
   - [x] e2e `options.mjs` and `sound.mjs`

## Later
- Chip styles, if still wanted once the rest is in.
