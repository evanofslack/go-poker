// Two-player session: hidden cards, host, next hand, reload, turn timer and grace period.
import { URL, check, finish, launch, shot } from "./lib.mjs";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// every frame a page receives, parsed
function record(page) {
    const rec = { games: [], logs: [], errors: [] };
    page.on("websocket", (ws) => {
        ws.on("framereceived", ({ payload }) => {
            try {
                const m = JSON.parse(payload);
                if (m.action === "update-game") rec.games.push(m.game);
                if (m.action === "new-log") rec.logs.push(m.message);
                if (m.action === "error") rec.errors.push(m.message);
            } catch {}
        });
    });
    return rec;
}

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
}

async function sit(page, seat) {
    await page.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await page.locator("#buyIn ~ button").first().click();
}

const lastGame = (rec) => rec.games[rec.games.length - 1];
const cardsShown = (g, name) => g.players.find((p) => p.username === name)?.cards[0] !== 0;

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
const preShowdown = (rec) => rec.games.filter((g) => g.running && g.stage !== 1);
check(
    "alice never receives bob's cards mid-hand",
    preShowdown(aRec).every((g) => !cardsShown(g, "bob")) && cardsShown(lastGame(aRec), "alice"),
);
check(
    "bob never receives alice's cards mid-hand",
    preShowdown(bRec).every((g) => !cardsShown(g, "alice")) && cardsShown(lastGame(bRec), "bob"),
);

// only the host sees reset
const hostMenu = 'button[aria-haspopup=menu]:has-text("host")';
check("host sees the host menu", (await alice.locator(hostMenu).count()) === 1);
check("non-host doesn't see it", (await bob.locator(hostMenu).count()) === 0);

// alice (dealer, small blind) folds; the server logs it and deals the next hand once
await alice.click('button:text-is("fold")');
await waitFor(() => aRec.logs.filter((l) => l === "starting new hand").length >= 2, 8000, "next hand");
await sleep(1000);
const hands = aRec.logs.filter((l) => l === "starting new hand").length;
check("server deals the next hand exactly once", hands === 2, `logs: ${JSON.stringify(aRec.logs)}`);
check("fold and win logged by the server", aRec.logs.includes("alice folds") && aRec.logs.includes("bob wins 30"));

// reload bob's tab mid-hand: it rejoins and keeps the seat
const bobStack = lastGame(aRec).players.find((p) => p.username === "bob").stack;
await bob.reload();
const rejoined = await waitFor(
    () => bRec.games.length && lastGame(bRec).players.some((p) => p.username === "bob") && cardsShown(lastGame(bRec), "bob"),
    5000,
    "bob to rejoin",
);
await sleep(500);
check("reload reclaims the seat with its cards", rejoined && (await bob.locator("#username").count()) === 0);
check("stack kept after reload", lastGame(aRec).players.find((p) => p.username === "bob").stack === bobStack);
await shot(bob, "4-bob-after-reload");

// close bob's tab; bob acts first in hand 2 (dealer), so the turn timer acts for him
await bobCtx.close();
console.log("  waiting for the 30s turn timer...");
const timedOut = await waitFor(() => aRec.logs.some((l) => l.startsWith("bob") && l.includes("(out of time)")), 40000, "turn timeout");
check("turn timer acts for a closed tab", timedOut, aRec.logs.slice(-6).join(" | "));
console.log("  waiting for the 60s grace period...");
const removed = await waitFor(() => aRec.logs.includes("bob left the table"), 70000, "bob removed");
check("closed tab is removed after the grace period", removed);
await shot(alice, "5-alice-after-bob-left");

await browser.close();
finish();
