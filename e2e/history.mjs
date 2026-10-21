// Hands tab: the hands played, with only your own and shown cards.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const openHands = async (p) => {
    await p.click('header button:has-text("table")');
    await p.click('[aria-label="table"] button:text-is("hands")');
};
const hand = (p, n) => p.locator('[aria-label="hand history"] > li').filter({ has: p.locator(`button[aria-label="hand ${n}"]`) });
// cards shown in a hand's player list, once it's open
const cardsIn = async (p, n) => {
    const li = hand(p, n);
    if ((await li.locator("button[aria-expanded=false]").count()) > 0) await li.locator("button").first().click();
    return li.locator('ul[aria-label="players"] [role=img]').count();
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "hist" + Date.now();
for (const [p, name, btn, seat] of [[alice, "alice", "create", 1], [bob, "bob", "join", 2]]) {
    await p.goto(URL);
    await p.fill("#username", name);
    await p.fill("#tablename", table);
    await p.click(`button:text-is("${btn}")`);
    if (btn === "create") await confirmCreate(p);
    await p.waitForSelector("text=Open");
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(300);
}
await openHands(alice);
check("no hands before the first", await has(alice, "text=hands show up here once they're played"));

// hand 1: alice, first to act, folds
await alice.click('button:text-is("start")');
await alice.click('button:text-is("fold"):enabled', { timeout: 5000 });
check("the hand shows up", await has(alice, 'button[aria-label="hand 1"]'));

// hand 2: check or call down to showdown
await Promise.any([alice, bob].map((p) => p.waitForSelector('button:text-matches("^call \\\\d+$")', { timeout: 15000 })));
// streets are dealt a second after betting closes
for (let i = 0; i < 40 && !(await alice.locator('button[aria-label="hand 2"]').count()); i++) {
    await alice.waitForTimeout(150);
    for (const p of [alice, bob]) {
        const b = p.locator('button:text-matches("^(check|call \\\\d+)$"):enabled');
        if (await b.count()) {
            await b.first().click().catch(() => {});
            await p.waitForTimeout(250);
        }
    }
}
check("the showdown shows up", await has(alice, 'button[aria-label="hand 2"]'));
check("newest hand starts open", (await hand(alice, 2).locator("button").first().getAttribute("aria-expanded")) === "true");
check("showdown shows both hands", (await cardsIn(alice, 2)) === 4);
const pot = await hand(alice, 2).locator('ul[aria-label="pots"]').innerText();
check("pot names the hand with its ranks", /pot \d+ · .+, .*(s|high|flush)$/.test(pot.trim()), pot);
check("streets run to the river", (await hand(alice, 2).locator("p", { hasText: /^river/ }).count()) === 1);
await shot(alice, "history");

await openHands(bob);
check("bob sees only his own cards from the fold", (await cardsIn(bob, 1)) === 2);
check("so does alice", (await cardsIn(alice, 1)) === 2);

// a late joiner sees shown cards only, and a reload keeps the hands
const carol = await (await browser.newContext()).newPage();
await carol.goto(URL);
await carol.fill("#username", "carol");
await carol.fill("#tablename", table);
await carol.click('button:text-is("join")');
await carol.waitForSelector("text=Open");
await openHands(carol);
check("a late joiner gets the hands", await has(carol, "text=2 hands"));
check("with shown cards", (await cardsIn(carol, 2)) === 4);
check("and none from the fold", (await cardsIn(carol, 1)) === 0);
await carol.check("text=hands I played");
check("filtering to your hands", await has(carol, "text=you haven't played a hand yet"));

await bob.reload();
await bob.waitForSelector('button[aria-label^="chat"]');
await openHands(bob);
check("a reload keeps the hands", await has(bob, "text=2 hands"));
check("and your own folded-against cards", (await cardsIn(bob, 1)) === 2);
await bob.click('[aria-label="close table panel"]');
await bob.click('button[aria-label^="chat"]');
await bob.click('button:text-is("log")');
check("a reload keeps the log", await has(bob, "text=alice sits down with"));
check("without repeating lines", (await bob.locator("text=alice sits down with").count()) === 1);
check("and no poker actions", (await bob.locator("text=alice folds").count()) === 0);

await alice.setViewportSize({ width: 360, height: 740 });
await shot(alice, "history-phone");
const close = await alice.locator('[aria-label="close table panel"]').boundingBox();
check("the close button stays on a phone", close && close.x + close.width <= 360 - 8, JSON.stringify(close));
await browser.close();
finish();
