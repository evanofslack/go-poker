# Table polish

**Status: in progress.** Stage 13 in `todo.md`.

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
3. **gliding buttons**: one dealer button and one marker per blind or straddle, drawn by the
   table rather than by each seat, so they glide to the next seats each hand. Phones keep them on
   the seat plates.
4. **stack changes and turn pulse**: stacks and the pot count to their new value; the seat to act
   pulses once as its turn starts.
5. **display tab**: a `prefs` object in the store (one localStorage key, merged with defaults, one
   `setPref` action), with the keyboard shortcuts toggle moved into it, out of the settings tab.
   Options, each a data attribute set before first paint by a small script in `_document.tsx`:
   - [ ] animations: system, on or off
   - [ ] deck: four colors (now) or two
   - [ ] table color: presets for the felt and rail tokens
   - [ ] card backs: presets for `card-back`
   - [ ] e2e `display.mjs`: each option sets its attribute and survives a reload

## Later
- Light mode: Mantine's color scheme instead of forced dark, light values for the role tokens, an
  audit of hard-coded colors (`ring-black/20`, the seat glows), screenshots of every panel and
  dialog. It covers the home page too, so it needs a toggle outside the table panel as well.
- Chip styles, if still wanted once the rest is in.
