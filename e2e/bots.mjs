// Bots: the host adds bots of each level, plays a hand against them, removes one, and starts
// a game of only bots without taking a seat.
import { setTimeout as sleep } from "node:timers/promises";
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const names = ["harper", "eric", "yasmin", "robert", "gus", "rishi", "venetia", "kenny", "daria", "sweetpea", "henry", "jesse"];

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
const handsWon = (rec) => rec.hands.length;
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
check("bots default to medium, with an icon", await alice.waitForSelector('p:has-text("harper") >> [data-bot-badge][title="medium bot"]', { timeout: 3000 }).then(() => true).catch(() => false));

await alice.locator("button", { hasText: "+ bot" }).filter({ hasText: "3" }).click();
const second = await alice.inputValue('input[maxlength="20"]');
check("suggested name skips taken ones", second !== "harper", second);
await alice.click('label:has-text("easy")');
await alice.click('button:text-is("add bot")');
await waitFor(() => lastGame(aRec)?.players.length === 3, 3000, "second bot");
const levels = () => (lastGame(aRec)?.players ?? []).map((p) => p.bot).filter(Boolean).sort();
check("server sends each bot's level", JSON.stringify(levels()) === '["easy","medium"]', JSON.stringify(levels()));
check("easy bot shows its level on hover", await alice.locator('[data-seat] [title="easy bot"]').count() === 1);
check("seats don't spell out bot levels", await alice.locator("[data-seat] >> text=/(easy|medium|hard) bot/").count() === 0);

// the host's remove buttons sit clear of the badges
const clash = await alice.evaluate(() => {
    const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
    const badges = [...document.querySelectorAll("[data-bot-badge]")].map((e) => e.getBoundingClientRect());
    return [...document.querySelectorAll('button[title="remove bot"]')].some((x) => badges.some((b) => hit(x.getBoundingClientRect(), b)));
});
check("remove buttons don't cover the badges", !clash);

// on a phone the icon is whole, with the level on hover
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
    "phone bot icons are whole",
    badges.length === 2 && badges.every((b) => b.text === "" && b.whole && / bot$/.test(b.title)),
    JSON.stringify(badges)
);
await shot(phone, "bots-phone");
await phone.context().close();

// the players panel badges bots, with their buy-in in the ledger
await alice.click('header button:has-text("table")');
await alice.click('[aria-label="table"] button:text-is("players")');
const harperRow = await alice.locator('ul[aria-label="players"] li').filter({ hasText: "harper" }).innerText();
check("players panel badges bots, with their buy-in", harperRow.includes("medium bot") && /in \d+/.test(harperRow), harperRow);
await alice.click('button[aria-label="manage harper"]');
const offered = await alice.locator("[role=dialog] button").allInnerTexts();
check("bots can't be made host, banker or a watcher", !offered.some((t) => /make host|make banker|move to watching/.test(t)), offered.join(", "));
await alice.keyboard.press("Escape");
await alice.click('button[aria-label="close table panel"]');

// alice checks or calls on her turns until a hand finishes
await alice.click('button:text-is("start")');
const end = Date.now() + 30000;
while (handsWon(aRec) < 1 && Date.now() < end) {
    const call = alice.locator('button:text-matches("^(check|call \\\\d+|all in \\\\d+)$")');
    if (await call.count()) await call.first().click().catch(() => {});
    await sleep(200);
}
check("a hand against bots finishes", handsWon(aRec) >= 1, JSON.stringify(aRec.hands.at(-1)?.actions));
check(
    "bots act on their own",
    aRec.hands.some((h) => h.actions.some((a) => a.name === "harper" && !a.kind.endsWith("blind"))),
);
await shot(alice, "bots-2-hand");

await alice.click('button[aria-label="remove harper"]');
await waitFor(() => !lastGame(aRec)?.players.some((p) => p.name === "harper"), 10000, "harper removed");
check("host removes a bot", !lastGame(aRec)?.players.some((p) => p.name === "harper"));

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
// bots think 1-2s per move, streets pause 1s and hands are 5s apart, so two hands take over a minute
check("bots-only game plays hands", await waitFor(() => handsWon(bRec) >= 2, 90000, "two bot hands"));
check("spectating host sees no bot cards mid-hand", bRec.games.filter((g) => g.betting && g.street !== "preflop").every((g) => g.players.every((p) => p.cards[0] === 0) || g.players.every((p) => !p.in || p.stack === 0)));
await shot(bob, "bots-3-alone");

check("no errors", aRec.errors.length === 0 && bRec.errors.length === 0, JSON.stringify([...aRec.errors, ...bRec.errors]));
await browser.close();
finish();
