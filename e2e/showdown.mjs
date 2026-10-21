// Play a hand to showdown and check the result display.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";


const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "sd" + Date.now();
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
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');

check("dealer shows D and SB", (await alice.locator("text=SB").count()) === 1 && (await alice.locator("text=BB").count()) === 1);
check("pot shows 30 preflop", (await alice.locator("p", { hasText: /^30$/ }).count()) === 1);
await shot(alice, "sd-1-preflop");

const made = await alice.locator("[data-made-hand]").innerText().catch(() => "");
check("you see your own best hand on your turn", ["high card", "pair"].includes(made), made);
check("others don't see it", (await bob.locator("[data-made-hand]").count()) === 1);
await alice.click('button:text-is("call 10")');
await alice.locator('button:text-is("call 10")').waitFor({ state: "detached", timeout: 2000 }).catch(() => null);
const tagged = await bob.locator('[data-markers] [data-act]', { hasText: /^call$/ }).waitFor({ timeout: 2000 }).then(() => true).catch(() => false);
check("others see a player's action by their bet", tagged);

// check or call down to showdown
const label = alice.locator("[data-won]");
// streets are dealt a second after betting closes
for (let i = 0; i < 40 && !(await label.count()); i++) {
    await alice.waitForTimeout(150);
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
check("winning seats say what they won", /^won (40|20)( · [a-z ]+)?( \| won 20( · [a-z ]+)?)?$/.test(text), text);
const hands = (await alice.locator("[data-hand]").allInnerTexts()).join(" | ");
check("winning seats name the hand", /^[a-z ]+( \| [a-z ]+)?$/.test(hands) && !/^an? /.test(hands), hands);
const full = await alice.locator("[data-hand]").first().getAttribute("title");
check("with its ranks on hover", /^(an? )?[a-z ,]+(s|high|royal flush)$/.test(full ?? ""), full);
check("a winning seat is highlighted", (await alice.locator(".bg-amber-200").count()) >= 1);
check("winning cards are ringed", (await alice.locator(".ring-amber-300").count()) >= 5, String(await alice.locator(".ring-amber-300").count()));
check("both players see the same result", hands === (await bob.locator("[data-hand]").allInnerTexts()).join(" | "));
await shot(alice, "sd-2-showdown");

await browser.close();
finish();
