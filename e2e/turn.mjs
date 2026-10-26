// Turn countdown and background-tab alert.
import { check, finish, join, launch, open, quiet, seat, shot, sit, until } from "./lib.mjs";

const browser = await launch();
const table = "turn" + Date.now();
const [alice] = await seat(browser, table, ["alice"]);
// bob's tab is in the background
const bob = await open(browser);
await bob.addInitScript(() => Object.defineProperty(document, "hidden", { get: () => true }));
await join(bob, "bob", table);
await sit(bob, 2);
const title = await bob.title();
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');
const width = () => alice.evaluate(() => document.querySelector('div[style*="width"]')?.style.width);
const w1 = await width();
await quiet(1500);
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
const flashed = () => bob.evaluate((title) => window.titles.includes("● your turn") && window.titles.includes(title), title);
check("background tab title flashes on your turn", await until(flashed, 5000), JSON.stringify(await bob.evaluate(() => window.titles)));
check("mute toggle is shown", (await alice.locator('button[title="mute"]').count()) === 1);
check("cards have names", (await alice.locator('[role=img][aria-label$=" of hearts"], [role=img][aria-label$=" of spades"], [role=img][aria-label$=" of clubs"], [role=img][aria-label$=" of diamonds"]').count()) === 2);
check("opponent cards read as face down", (await alice.locator('[role=img][aria-label="face-down card"]').count()) === 2);
check("the player to act is named for screen readers", (await alice.locator('.sr-only:text-is("to act")').count()) === 1);
await browser.close();
finish();
