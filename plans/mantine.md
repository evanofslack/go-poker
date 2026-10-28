# Mantine

**Status: in progress.** Stage 12 in `todo.md`.

Move the web app's controls to Mantine 8 and give them one style guide (in `CLAUDE.md`, under UI).
Before this, `Button` had four variants whose fills didn't follow a rule, every call site set its
own padding, and about 25 raw `<button>`s each invented a style. Mantine gives a fixed vocabulary,
focus traps and menu keys we didn't have, and light mode later.

## Decisions
- Mantine for controls and overlays; Tailwind stays for layout, the table, seats, cards and the
  `phone:`/`tall:`/`short:` variants. Responsive sizes stay in `className`.
- Use Mantine components directly. Defaults live in the theme, so call sites say what differs.
- One palette: the theme holds the colors (zinc as `dark` and `gray`, Tailwind's cyan, rose as
  `red`); the Tailwind role tokens point at `--mantine-color-*`. Dark is forced for now.
- CSS layers `theme, base, mantine, components, utilities`: Mantine beats the preflight, utility
  classes beat Mantine.
- The betting buttons are all `default`: none is the main action.
- `Hint` keeps expanding inline (works on touch), as an `ActionIcon`.
- Mantine `Slider` for the raise amount. 1px borders, Mantine's default.

## PRs
Stacked. Each runs `just check` and `just e2e`, with screenshots.

1. **mantine setup and buttons**: provider, theme, layers, tokens on Mantine vars; every `Button`
   and the raw icon buttons on Mantine; the style guide applied (fold neutral, start filled, one
   filled per group, stand-up start and cancel left); e2e's `btn(text)`.
2. **mantine controls and overlays**: `Tabs` for chat/log and the table panel,
   `SegmentedControl`, `Switch`, `Checkbox`; `Modal` under `Dialog` (no transition, no close
   button, so focus and e2e work as before); `Menu` for `SeatMenu`; `@mantine/notifications`
   for `Toaster`, with a `toast()` helper replacing the store's toasts.
3. **mantine inputs and color tokens**: `TextInput`, `PasswordInput`, `NumberInput` (keep every
   `id`, e2e presses them); `Slider` and `NumberInput` in `RaiseInput` (its arrow handling skips
   `role="slider"`); raise presets as `Button`s; the hard-coded colors in `Table`, `Card`, `Seat`,
   `History`, `Pot` and others as tokens, so light mode is light values plus a toggle.

## Notes
- Mantine's dark scheme takes outline and light colors 4-5 shades below `primaryShade`, which
  lands near white in Tailwind's palettes; `resolver` in `lib/theme.ts` uses the 400s.
- `SegmentedControl` doesn't wrap: check the six-option rows (stakes, hand delay) on a phone,
  and use `Chip.Group` if they don't fit.
- Playwright's `:text-is` only sees an element's own text; `btn(text)` also matches a button
  holding it, as Mantine's do.
