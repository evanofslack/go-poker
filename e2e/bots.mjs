// Bots: the host adds bots of each level, plays a hand against them, removes one, and starts
// a game of only bots without taking a seat.
import { setTimeout as sleep } from "node:timers/promises";
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const names = ["harper", "eric", "yasmin", "robert", "gus", "rishi", "venetia", "kenny", "daria", "sweetpea", "henry", "jesse"];

function record(page) {
    const rec = { games: [], logs: [], errors: [], bots: {} };
    page.on("websocket", (ws) => {
        ws.on("framereceived", ({ payload }) => {
            try {
                const m = JSON.parse(payload);
                if (m.action === "update-game") {
                    rec.games.push(m.game);
                    rec.bots = m.bots;
                }
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

async function create(page, name, table) {
    await page.goto(URL);
    await page.fill("#username", name);
    await page.fill("#tablename", table);
    await page.click('button:text-is("create")');
    await confirmCreate(page);
    await page.waitForSelector("text=Open");
}

const lastGame = (rec) => rec.games[rec.games.length - 1];
const handsWon = (rec) => rec.logs.filter((l) => / (wins|split) /.test(l)).length;
const browser = await launch();

// a seated host adds two bots from the empty seats
const alice = await (await browser.newContext()).newPage();
const aRec = record(alice);
await create(alice, "alice", "bots" + Date.now());
await alice.locator("button", { hasText: "Open" }).filter({ hasText: "1" }).click();
await alice.locator("#buyIn ~ button").first().click();
await alice.waitForSelector("text=+ bot");

await alice.locator("button", { hasText: "+ bot" }).filter({ hasText: "2" }).click();
const suggested = await alice.inputValue('input[maxlength="20"]');
check("bot name is suggested from the list", names.includes(suggested), suggested);
await alice.fill('input[maxlength="20"]', "harper");
await shot(alice, "bots-1-dialog");
await alice.click('button:text-is("add bot")');
check("bots default to medium, with a badge", await alice.waitForSelector('p:has-text("harper") >> text=medium bot', { timeout: 3000 }).then(() => true).catch(() => false));

await alice.locator("button", { hasText: "+ bot" }).filter({ hasText: "3" }).click();
const second = await alice.inputValue('input[maxlength="20"]');
check("suggested name skips taken ones", second !== "harper", second);
await alice.click('label:has-text("easy")');
await alice.click('button:text-is("add bot")');
await waitFor(() => lastGame(aRec)?.players.length === 3, 3000, "second bot");
check("server sends each bot's level", JSON.stringify(Object.values(aRec.bots).sort()) === '["easy","medium"]', JSON.stringify(aRec.bots));
check("easy bot shows its level", await alice.locator("text=easy bot").count() === 1);

// the host's remove buttons sit clear of the badges
const clash = await alice.evaluate(() => {
    const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
    const badges = [...document.querySelectorAll("[data-bot-badge]")].map((e) => e.getBoundingClientRect());
    return [...document.querySelectorAll('button[title="remove bot"]')].some((x) => badges.some((b) => hit(x.getBoundingClientRect(), b)));
});
check("remove buttons don't cover the badges", !clash);

// on a phone the badge is just "bot", whole, with the level on hover
const phone = await (await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true })).newPage();
await phone.goto(URL);
await phone.fill("#username", "pat");
await phone.fill("#tablename", new globalThis.URL(alice.url()).searchParams.get("table"));
await phone.click('button:text-is("join")');
await phone.waitForSelector("[data-bot-badge]");
const badges = await phone.evaluate(() =>
    [...document.querySelectorAll("[data-bot-badge]")].map((e) => ({
        text: e.innerText.trim(),
        title: e.title,
        whole: e.getBoundingClientRect().right <= e.parentElement.getBoundingClientRect().right + 1,
    }))
);
check(
    "phone bot badges are short and whole",
    badges.length === 2 && badges.every((b) => b.text === "bot" && b.whole && / bot$/.test(b.title)),
    JSON.stringify(badges)
);
await shot(phone, "bots-phone");
await phone.context().close();

// the players panel badges bots, with their buy-in in the ledger
await alice.click('button[aria-label^="chat"]');
await alice.click('button:text-is("players")');
const harperRow = await alice.locator('ul[aria-label="players"] li').filter({ hasText: "harper" }).innerText();
check("players panel badges bots, with their buy-in", harperRow.includes("medium bot") && /in \d+/.test(harperRow), harperRow);
await alice.click('button[aria-label="manage harper"]');
const offered = await alice.locator("[role=dialog] button").allInnerTexts();
check("bots can't be made host, banker or a watcher", !offered.some((t) => /make host|make banker|move to watching/.test(t)), offered.join(", "));
await alice.keyboard.press("Escape");
await alice.click('button[aria-label="close chat"]');

// alice checks or calls on her turns until a hand finishes
await alice.click('button:text-is("start")');
const end = Date.now() + 20000;
while (handsWon(aRec) < 1 && Date.now() < end) {
    const call = alice.locator('button:text-matches("^(check|call \\\\d+|all in \\\\d+)$")');
    if (await call.count()) await call.first().click().catch(() => {});
    await sleep(200);
}
check("a hand against bots finishes", handsWon(aRec) >= 1, JSON.stringify(aRec.logs.slice(-8)));
check("bots act on their own", aRec.logs.some((l) => /^harper (checks|calls|raises|bets|folds)/.test(l)));
await shot(alice, "bots-2-hand");

await alice.click('button[aria-label="remove harper"]');
await waitFor(() => !lastGame(aRec)?.players.some((p) => p.username === "harper"), 10000, "harper removed");
check("host removes a bot", !lastGame(aRec)?.players.some((p) => p.username === "harper"));

// a host without a seat runs a game of only bots
const bob = await (await browser.newContext()).newPage();
const bRec = record(bob);
await create(bob, "bob", "botsonly" + Date.now());
for (const seat of ["1", "2"]) {
    await bob.locator("button", { hasText: "Open" }).filter({ hasText: seat }).click();
    await bob.click('button:text-is("add a bot here instead")');
    await bob.click('button:text-is("add bot")');
    await sleep(300);
}
await bob.click('button:text-is("start")');
// bots think 1-2s per move and hands are 5s apart, so two hands take up to a minute
check("bots-only game plays hands", await waitFor(() => handsWon(bRec) >= 2, 60000, "two bot hands"));
check("spectating host sees no bot cards mid-hand", bRec.games.filter((g) => g.betting && g.stage > 1).every((g) => g.players.every((p) => p.cards[0] === 0) || g.players.every((p) => !p.in || p.stack === 0)));
await shot(bob, "bots-3-alone");

check("no errors", aRec.errors.length === 0 && bRec.errors.length === 0, JSON.stringify([...aRec.errors, ...bRec.errors]));
await browser.close();
finish();
