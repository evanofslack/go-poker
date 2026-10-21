# e2e

Browser tests that drive real players through the static build: hidden cards, host controls, next hand,
reload and reconnect, turn timer (`session.mjs`), betting controls, showdown display, turn countdown,
seat menu, the join and buy-in forms, password tables, the players panel, host controls, invite links, toasts, reconnecting, the header, leaving and the host menu (`ux.mjs`), nine
seats on desktop, tablet and phone screens (`layout.mjs`), and playing with bots (`bots.mjs`). They aren't run in CI.

- `cd e2e && npm install`
- `npx playwright-core install chromium`, or point `CHROMIUM_PATH` at a Chromium you already have
- `just e2e` builds the web app and server, runs the server on :8099 and runs every script.
  Set `SHOTS=<dir>` to save screenshots.

`session.mjs` takes about 100s, since it waits on the real 30s turn timer and 60s grace period.
