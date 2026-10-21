// Play a hand to showdown and check the result display.
import { URL, check, finish, launch, shot } from "./lib.mjs";


const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "sd" + Date.now();
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
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');

check("dealer shows D and SB", (await alice.locator("text=SB").count()) === 1 && (await alice.locator("text=BB").count()) === 1);
check("pot shows 30 preflop", (await alice.locator("p", { hasText: /^30$/ }).count()) === 1);
await shot(alice, "sd-1-preflop");

// check or call down to showdown
const label = alice.locator("p", { hasText: /pot \d+ · / });
for (let i = 0; i < 12 && !(await label.count()); i++) {
    for (const p of [alice, bob]) {
        const b = p.locator('button:text-matches("^(check|call \\\\d+)$")');
        if (await b.count()) {
            await b.first().click();
            await p.waitForTimeout(250);
        }
    }
}
await label.first().waitFor({ timeout: 3000 }).catch(() => null);
const text = (await label.allInnerTexts()).join(" | ");
check("pot label names the winner and hand", /pot 40 · (split )?(alice|bob)(, bob)?, [a-z ]+$/.test(text), text);
check("a winning seat is highlighted", (await alice.locator(".bg-amber-200").count()) >= 1);
check("winning cards are ringed", (await alice.locator(".ring-amber-300").count()) >= 5, String(await alice.locator(".ring-amber-300").count()));
check("both players see the same result", text === (await bob.locator("p", { hasText: /pot \d+ · / }).allInnerTexts()).join(" | "));
await shot(alice, "sd-2-showdown");

await browser.close();
finish();
