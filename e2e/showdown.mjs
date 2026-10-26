// Play a hand to showdown and check the result display.
import { check, checkDown, finish, gone, launch, seat, shot } from "./lib.mjs";

const browser = await launch();
const [alice, bob] = await seat(browser, "sd" + Date.now(), ["alice", "bob"]);
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');

check("dealer shows D and SB", (await alice.locator("text=SB").count()) === 1 && (await alice.locator("text=BB").count()) === 1);
check("pot shows 30 preflop", (await alice.locator("p", { hasText: /^30$/ }).count()) === 1);
await shot(alice, "sd-1-preflop");

const made = await alice.locator("[data-made-hand]").innerText().catch(() => "");
check("you see your own best hand on your turn", ["high card", "pair"].includes(made), made);
check("others don't see it", (await bob.locator("[data-made-hand]").count()) === 1);
await alice.click('button:text-is("call 10")');
const tagged = await bob.locator("[data-markers] [data-act]", { hasText: /^call$/ }).waitFor({ timeout: 2000 }).then(() => true, () => false);
check("others see a player's action by their bet", tagged);

// the big blind checks their option and the flop comes
await bob.click('button:text-is("check")');
check("blind labels go after preflop", (await gone(alice, "text=SB")) && (await alice.locator("text=BB").count()) === 0);

const won = alice.locator("[data-won]");
await checkDown([alice, bob], async () => (await won.count()) > 0);
const text = (await won.allInnerTexts()).join(" | ");
check("winning seats say what they won", /^won \d+/.test(text), text);
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
