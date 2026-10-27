// Run it: two players all in preflop vote on how many times to run the board out.
import { check, finish, has, launch, panel, seat, shot, until } from "./lib.mjs";

const setting = (p, label) => p.locator('dl[aria-label="table settings"] > div', { hasText: label }).innerText();
const runIt = (p) => p.click("#runIt");

// alice, on the button, shoves and bob calls
async function allIn(alice, bob) {
    await alice.click('button:text-is("start")');
    await alice.waitForSelector('button:text-is("call 10")');
    await alice.click('button:text-is("raise")');
    await alice.click('button:text-is("all in")');
    await alice.press("#raise", "Enter");
    await bob.getByRole("button", { name: /^(call|all in) \d+/ }).click();
}

const browser = await launch();
const [alice, bob] = await seat(browser, "ri" + Date.now(), ["alice", "bob"], runIt);

await panel(alice, "settings");
const shown = await setting(alice, "run it");
check("settings show run it", /up to 4 times$/.test(shown.trim()), shown);
await alice.click('[aria-label="close table panel"]');

await allIn(alice, bob);
check("both are asked how many times", (await has(alice, "[data-run-ask]")) && (await has(bob, "[data-run-ask]")));
const options = await bob.locator("[data-run-ask] button").allInnerTexts();
check("from once to 4", JSON.stringify(options) === '["once","2×","3×","4×"]', JSON.stringify(options));
await alice.click('[data-run-ask] button:text-is("2×")');
check("alice's pick is marked", (await alice.locator('[data-run-ask] button[aria-pressed="true"]').innerText()) === "2×");
check("the board waits for bob", (await alice.locator("[data-board] [role=img]").count()) === 0);
await shot(bob, "ri-1-vote");
await bob.click('[data-run-ask] button:text-is("3×")');

check("the smallest pick runs: two boards", await has(alice, '[data-boards="2"]', 3000));
check("the prompt goes", await until(async () => (await alice.locator("[data-run-ask]").count()) === 0));
const settled = await until(async () => (await alice.locator("[data-run-caption]", { hasText: "·" }).count()) === 2, 15000);
const captions = await alice.locator("[data-run-caption]").allInnerTexts();
check("each run names its winner", settled && /^run 1 · /.test(captions[0]) && /^run 2 · /.test(captions[1]), JSON.stringify(captions));
check("the second run is dealt in full", (await alice.locator('[data-run="2"] [role=img]').count()) === 5);
check("both see the same runs", JSON.stringify(captions) === JSON.stringify(await bob.locator("[data-run-caption]").allInnerTexts()));
await shot(alice, "ri-2-two-boards");

await panel(alice, "hands"); // the newest hand is open
check("history shows the second board", await has(alice, "[data-history-run]"));
const pots = await alice.locator('ul[aria-label="pots"] li').allInnerTexts();
check("and a pot for each run", pots.length === 2 && /^run 1: pot/.test(pots[0]) && /^run 2: pot/.test(pots[1]), JSON.stringify(pots));
await shot(alice, "ri-3-history");
await alice.click('[aria-label="close table panel"]');

// a pick of once ends the vote
const [carol, dan] = await seat(browser, "ri1" + Date.now(), ["carol", "dan"], runIt);
await allIn(carol, dan);
await dan.click('[data-run-ask] button:text-is("once")');
check("once ends the vote", await until(async () => (await carol.locator("[data-run-ask]").count()) === 0));
check("with one board", (await carol.locator('[data-boards="1"]').count()) === 1 && (await carol.locator("[data-run]").count()) === 0);

// without run it, nobody is asked
const [erin, finn] = await seat(browser, "ri0" + Date.now(), ["erin", "finn"]);
await allIn(erin, finn);
check("no vote without run it", !(await has(erin, "[data-run-ask]", 1500)));

await browser.close();
finish();
