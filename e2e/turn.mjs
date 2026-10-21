// Turn countdown and background-tab alert.
import { URL, check, finish, launch, shot } from "./lib.mjs";
const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
// bob's tab is in the background
await bob.addInitScript(() => Object.defineProperty(document, "hidden", { get: () => true }));
const table = "turn" + Date.now();
for (const [p, name, btn, seat] of [[alice, "alice", "create", 1], [bob, "bob", "join", 2]]) {
    await p.goto(URL);
    await p.fill("#username", name);
    await p.fill("#tablename", table);
    await p.click(`button:text-is("${btn}")`);
    await p.waitForSelector("text=Open");
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(300);
}
const title = await bob.title();
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');
const bar = alice.locator(".bg-accent, .bg-danger").filter({ has: alice.locator("xpath=.") });
const width = () => alice.evaluate(() => document.querySelector('div[style*="width"]')?.style.width);
const w1 = await width();
await alice.waitForTimeout(1500);
const w2 = await width();
check("countdown shows on the active seat", !!w1, w1);
check("countdown counts down", parseFloat(w2) < parseFloat(w1), `${w1} -> ${w2}`);
check("only one countdown", (await alice.locator('div[style*="width"]').count()) === 1, JSON.stringify(await alice.evaluate(() => [...document.querySelectorAll('div[style*="width"]')].map((e) => e.outerHTML.slice(0, 120)))));
await shot(alice, "turn-1-countdown");

// record every title bob's tab shows from here on
await bob.evaluate(() => {
    window.titles = [];
    new MutationObserver(() => window.titles.push(document.title)).observe(document.querySelector("title"), { childList: true, characterData: true, subtree: true });
});
await alice.click('button:text-is("call 10")');
await bob.waitForTimeout(3000);
const titles = await bob.evaluate(() => window.titles);
check("background tab title flashes on your turn", titles.includes("● your turn") && titles.includes(title), JSON.stringify(titles));
check("mute toggle is shown", (await alice.locator('button[title="mute"]').count()) === 1);
check("cards have names", (await alice.locator('[role=img][aria-label$=" of hearts"], [role=img][aria-label$=" of spades"], [role=img][aria-label$=" of clubs"], [role=img][aria-label$=" of diamonds"]').count()) === 2);
check("opponent cards read as face down", (await alice.locator('[role=img][aria-label="face-down card"]').count()) === 2);
check("the player to act is named for screen readers", (await alice.locator('.sr-only:text-is("to act")').count()) === 1);
await browser.close();
finish();
