// Betting controls: labels, presets, raise, and the waiting line.
import { btn, check, finish, has, launch, seat, shot } from "./lib.mjs";

const buttons = async (p) =>
    (await p.locator("button").allInnerTexts()).map((t) => t.trim()).filter((t) => /^(fold|check|call|bet|raise|all in)/.test(t));

const browser = await launch();
const [alice, bob] = await seat(browser, "bet" + Date.now(), ["alice", "bob"]);
await alice.click(btn("start"));
await alice.waitForSelector(btn("call 10"));

check("small blind sees fold / call 10 / raise", JSON.stringify(await buttons(alice)) === '["fold","call 10","raise"]', JSON.stringify(await buttons(alice)));
check("other player sees waiting line", (await bob.locator("text=waiting for alice").count()) === 1);

await alice.click(btn("raise"));
const val = () => alice.inputValue("#raise");
check("raise input starts at min raise", (await val()) === "40", await val());
await alice.click(btn("pot"));
check("pot preset is 60", (await val()) === "60", await val());
await alice.click(btn("all in"));
check("all in preset is the whole stack", (await val()) === "2000", await val());
await alice.fill("#raise", "5");
check("too-small amount is clamped", (await alice.locator(btn("raise to 40")).count()) === 1);
await shot(alice, "bet-1-raise-input");
await alice.fill("#raise", "60");
await alice.press("#raise", "Enter");

await has(bob, btn("call 40"));
check("big blind faces call 40 after raise to 60", JSON.stringify(await buttons(bob)) === '["fold","call 40","raise"]', JSON.stringify(await buttons(bob)));
await shot(bob, "bet-2-bob-facing-raise");
await bob.click(btn("call 40"));

// heads up, the big blind acts first after the flop
await has(bob, btn("check"));
check("flop opens with check / bet", JSON.stringify(await buttons(bob)) === '["fold","check","bet"]', JSON.stringify(await buttons(bob)));
await bob.click(btn("bet"));
await bob.click(btn("pot"));
check("pot bet on a 120 pot is 120", (await bob.inputValue("#raise")) === "120", await bob.inputValue("#raise"));
check("bet button says bet 120", (await bob.locator(btn("bet 120")).count()) === 1);
await bob.press("#raise", "Escape");
check("escape closes the raise input", (await bob.locator("#raise").count()) === 0);

await browser.close();
finish();
