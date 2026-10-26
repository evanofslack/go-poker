// House rules: antes, the 7-2 setting, showing cards after the hand, and duck hunting.
import { check, finish, gone, has, launch, panel, seat, shot } from "./lib.mjs";

// face-up cards in seats, not on the board
const faceUp = '[role=img]:not([aria-label="face-down card"]):not([aria-label$=", folded"])';
const setting = (p, label) => p.locator('dl[aria-label="table settings"] > div', { hasText: label }).innerText();
const count = (p, selector, n) =>
    p.waitForFunction(([s, n]) => document.querySelectorAll(s).length === n, [selector, n], { timeout: 3000 }).then(() => true, () => false);

// alice makes a table with a quarter big blind ante; duck hunting is on by default
const browser = await launch();
let huntDefault;
const [alice, bob] = await seat(browser, "hr" + Date.now(), ["alice", "bob"], async (p) => {
    huntDefault = await p.locator("#duckHunt").isChecked();
    await p.click('label:has(input[name="ante"]) >> text="25%"');
    await p.click('label:has(input[name="sevenDeuce"]) >> text="2 bb"');
});
check("duck hunt is allowed by default", huntDefault);

await panel(alice, "settings");
const ante = await setting(alice, "ante");
check("settings show the ante", /ante\s+5$/.test(ante.trim()), ante);
const hunt = await setting(alice, "duck hunt");
check("and duck hunting", /allowed$/.test(hunt.trim()), hunt);
const sevenDeuce = await setting(alice, "7-2");
check("and the 7-2 bounty", /40 each$/.test(sevenDeuce.trim()), sevenDeuce);
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
const before = await alice.locator(faceUp).count();
await bob.click('button:text-is("show cards")');
check(
    "alice sees bob's shown cards",
    await alice.waitForFunction(([s, n]) => document.querySelectorAll(s).length >= n + 2, [faceUp, before], { timeout: 3000 }).then(() => true, () => false)
);
check("the show button goes once shown", await gone(bob, 'button:text-is("show cards")'));

await alice.click('button:text-is("duck hunt")');
for (const [p, name] of [[alice, "alice"], [bob, "bob"]]) {
    check(`${name} sees the hunted board`, await count(p, "[data-hunted]", 5));
}
check("marked as a duck hunt", await has(bob, '[data-board] >> text="duck hunt"'));
check("only one hunt a hand", (await alice.locator('button:text-is("duck hunt")').count()) === 0);
await shot(alice, "houserules-hunt");

// the next hand clears the hunt
check("the next hand clears it", await gone(bob, "[data-hunted]", 10000));

await browser.close();
finish();
