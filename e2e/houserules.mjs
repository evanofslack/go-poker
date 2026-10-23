// House rules: antes, showing cards after the hand, and duck hunting.
import { URL, check, finish, launch, shot } from "./lib.mjs";

const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
// face-up cards in seats, not on the board
const faceUp = (p) => p.locator('[role=img]:not([aria-label="face-down card"]):not([aria-label$=", folded"])').count();

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "hr" + Date.now();

// alice makes a table with a quarter big blind ante; duck hunting is on by default
await alice.goto(URL);
await alice.fill("#username", "alice");
await alice.fill("#tablename", table);
await alice.click('button:text-is("create")');
await alice.click("text=more settings");
check("duck hunt is allowed by default", await alice.locator("#duckHunt").isChecked());
await alice.click('label:has(input[name="ante"]) >> text="25%"');
await alice.click('button:text-is("create table")');

await bob.goto(URL);
await bob.fill("#username", "bob");
await bob.fill("#tablename", table);
await bob.click('button:text-is("join")');
for (const [p, seat] of [[alice, 1], [bob, 2]]) {
    await p.waitForSelector("text=Open");
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(300);
}

await alice.click('header button:has-text("table")');
await alice.click('[aria-label="table"] button:text-is("settings")');
const ante = await alice.locator('dl[aria-label="table settings"] > div', { hasText: "ante" }).innerText();
check("settings show the ante", /ante\s+5$/.test(ante.trim()), ante);
const hunt = await alice.locator('dl[aria-label="table settings"] > div', { hasText: "duck hunt" }).innerText();
check("and duck hunting", /allowed$/.test(hunt.trim()), hunt);
await alice.click('[aria-label="close table panel"]');

// blinds of 10 and 20, and two antes of 5
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');
check("antes go in the pot", (await alice.locator("p", { hasText: /^40$/ }).count()) === 1);
check("no show or hunt mid-hand", (await alice.locator('[aria-label="after the hand"]').count()) === 0);

// alice folds: bob wins without showing
await alice.click('button:text-is("fold"):enabled');
check("bob can show his cards", await has(bob, 'button:text-is("show cards")'));
check("alice can duck hunt", await has(alice, 'button:text-is("duck hunt")'));
const before = await faceUp(alice);
await bob.click('button:text-is("show cards")');
check(
    "alice sees bob's shown cards",
    await alice.waitForFunction((n) => document.querySelectorAll('[role=img]:not([aria-label="face-down card"]):not([aria-label$=", folded"])').length >= n + 2, before, { timeout: 3000 }).then(() => true).catch(() => false)
);
check("the show button goes once shown", await bob.locator('button:text-is("show cards")').waitFor({ state: "detached", timeout: 2000 }).then(() => true).catch(() => false));

await alice.click('button:text-is("duck hunt")');
for (const [p, name] of [[alice, "alice"], [bob, "bob"]]) {
    check(`${name} sees the hunted board`, await p.waitForFunction(() => document.querySelectorAll("[data-hunted]").length === 5, null, { timeout: 3000 }).then(() => true).catch(() => false));
}
check("marked as a duck hunt", await has(bob, '[data-board] >> text="duck hunt"'));
check("only one hunt a hand", (await alice.locator('button:text-is("duck hunt")').count()) === 0);
await shot(alice, "houserules-hunt");

// the next hand clears the hunt
check("the next hand clears it", await bob.waitForFunction(() => document.querySelectorAll("[data-hunted]").length === 0, null, { timeout: 10000 }).then(() => true).catch(() => false));

await browser.close();
finish();
