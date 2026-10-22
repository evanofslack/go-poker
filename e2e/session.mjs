// Two-player session: hidden cards, next hand, reload, turn timer and grace period.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// every frame a page receives, parsed
function record(page) {
    const rec = { games: [], logs: [], errors: [], hands: [] };
    page.on("websocket", (ws) => {
        ws.on("framereceived", ({ payload }) => {
            try {
                const m = JSON.parse(payload);
                if (m.action === "welcome" || m.action === "table-update") rec.games.push(m.view);
                for (const e of m.events ?? []) {
                    if (e.type === "log") rec.logs.push(e.data.log.message);
                    // hands as they end
                    if (e.type === "hand-done") rec.hands.push(e.data.record);
                }
                if (m.action === "error") rec.errors.push(m.message);
            } catch {}
        });
    });
    return rec;
}

// dealt counts the hands dealt, from updates where betting starts
const dealt = (rec) => rec.games.filter((g, i) => g.betting && !rec.games[i - 1]?.betting).length;

async function waitFor(cond, ms, what) {
    const end = Date.now() + ms;
    while (Date.now() < end) {
        if (cond()) return true;
        await sleep(100);
    }
    console.log(`  timed out waiting for ${what}`);
    return false;
}

async function enter(page, name, table, button) {
    await page.goto(URL);
    await page.fill("#username", name);
    await page.fill("#tablename", table);
    await page.click(`button:text-is("${button}")`);
    if (button === "create") await confirmCreate(page);
}

async function sit(page, seat) {
    await page.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await page.locator("#buyIn ~ button").first().click();
}

const lastGame = (rec) => rec.games[rec.games.length - 1];
const cardsShown = (g, name) => g.players.find((p) => p.name === name)?.cards[0] !== 0;

const TABLE = "e2e" + Date.now();
const browser = await launch();
const aliceCtx = await browser.newContext();
const bobCtx = await browser.newContext();
const alice = await aliceCtx.newPage();
const bob = await bobCtx.newPage();
const aRec = record(alice);
const bRec = record(bob);

// create, join, sit, start
await enter(alice, "alice", TABLE, "create");
await alice.waitForSelector("text=Open");
await sit(alice, 1);
await waitFor(() => lastGame(aRec)?.players.length === 1, 3000, "alice seated");
await enter(bob, "bob", TABLE, "join");
await bob.waitForSelector("text=Open");
await sit(bob, 2);
await waitFor(() => lastGame(aRec)?.players.length === 2, 3000, "bob seated");

// a third tab can't take a name that's in use
const eve = await (await browser.newContext()).newPage();
await enter(eve, "ALICE", TABLE, "join");
const nameErr = await eve.waitForSelector("text=name already taken", { timeout: 3000 }).catch(() => null);
check("duplicate name rejected on join form", !!nameErr);
await shot(eve, "1-name-taken");
await eve.context().close();

await alice.click('button:text-is("start")');
await waitFor(() => lastGame(aRec)?.betting && lastGame(bRec)?.betting, 3000, "hand to start");
await shot(alice, "2-alice-hand1");
await shot(bob, "3-bob-hand1");

// hidden cards: every frame before the first showdown
const preShowdown = (rec) => rec.games.filter((g) => g.running && g.street !== "");
check(
    "alice never receives bob's cards mid-hand",
    preShowdown(aRec).every((g) => !cardsShown(g, "bob")) && cardsShown(lastGame(aRec), "alice"),
);
check(
    "bob never receives alice's cards mid-hand",
    preShowdown(bRec).every((g) => !cardsShown(g, "alice")) && cardsShown(lastGame(bRec), "bob"),
);

// alice (dealer, small blind) folds; the server records it and deals the next hand once
await alice.click('button:text-is("fold")');
await waitFor(() => dealt(aRec) >= 2, 8000, "next hand");
await sleep(1000);
check("server deals the next hand exactly once", dealt(aRec) === 2, `dealt ${dealt(aRec)}`);
const first = aRec.hands[0];
check(
    "fold and win recorded by the server",
    first?.actions.at(-1)?.name === "alice" && first?.actions.at(-1)?.kind === "fold" && first?.pots[0]?.winners[0] === "bob",
    JSON.stringify(first),
);
check("the log leaves out poker actions", !aRec.logs.some((l) => /folds|wins|starting new hand|blind/.test(l)), JSON.stringify(aRec.logs));

// reload bob's tab mid-hand: it rejoins and keeps the seat
const bobStack = lastGame(aRec).players.find((p) => p.name === "bob").stack;
await bob.reload();
const rejoined = await waitFor(
    () => bRec.games.length && lastGame(bRec).players.some((p) => p.name === "bob") && cardsShown(lastGame(bRec), "bob"),
    5000,
    "bob to rejoin",
);
await sleep(500);
check("reload reclaims the seat with its cards", rejoined && (await bob.locator("#username").count()) === 0);
check("stack kept after reload", lastGame(aRec).players.find((p) => p.name === "bob").stack === bobStack);
await shot(bob, "4-bob-after-reload");

// close bob's tab; bob acts first in hand 2 (dealer), so the turn timer acts for him
await bobCtx.close();
console.log("  waiting for the 30s turn timer...");
const timedOut = await waitFor(
    () => aRec.hands.some((h) => h.actions.some((a) => a.name === "bob" && a.timeout)),
    40000,
    "turn timeout",
);
check("turn timer acts for a closed tab", timedOut, JSON.stringify(aRec.hands.at(-1)?.actions));
console.log("  waiting for the 60s grace period...");
const removed = await waitFor(() => aRec.logs.includes("bob left the table"), 70000, "bob removed");
check("closed tab is removed after the grace period", removed);
await shot(alice, "5-alice-after-bob-left");

await browser.close();
finish();
