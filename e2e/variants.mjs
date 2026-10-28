// Other games: pot limit Omaha and Omaha 5, their cards on each screen, the pot
// limit presets, and short deck with its J-6 bounty.
import { btn, check, checkDown, finish, has, launch, record, seat, shot } from "./lib.mjs";

const phone = { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 };
const small = { viewport: { width: 360, height: 740 }, isMobile: true, hasTouch: true };
const game = (name, limit) => async (p) => {
    await p.click(`fieldset:has(input[name="game"]) >> text="${name}"`);
    await p.click(`fieldset:has(input[name="limit"]) >> text="${limit}"`);
};
const faceUp = (p) => p.locator('[role=img][aria-label$=" of hearts"], [role=img][aria-label$=" of spades"], [role=img][aria-label$=" of clubs"], [role=img][aria-label$=" of diamonds"]').count();
const browser = await launch();

// PLO three handed: alice on a desktop, bob on a phone, carol on a small phone
const [alice, bob, carol] = await seat(browser, "plo" + Date.now(), ["alice", { name: "bob", context: phone }, { name: "carol", context: small }], game("Omaha", "pot limit"));
await alice.click(btn("start"));
await alice.waitForSelector(btn("call 20"));
check("the header names the game", (await alice.locator("text=PLO 10/20").count()) === 1);
check("four cards each", (await faceUp(alice)) === 4 && (await alice.locator('[role=img][aria-label="face-down card"]').count()) === 8, `${await faceUp(alice)} up`);
check("your hint follows the two card rule", (await alice.locator("[data-made-hand]").innerText()).length > 0);
await shot(alice, "plo-1-desktop");
await shot(bob, "plo-2-phone");
await shot(carol, "plo-3-small");

// under the gun: 30 in, 20 to call, so the pot is 70
await alice.click(btn("raise"));
const presets = async () => (await alice.locator("form button").allInnerTexts()).map((t) => t.trim()).filter((t) => /pot|min|all in/.test(t));
check("pot limit presets", JSON.stringify(await presets()) === '["min","½ pot","¾ pot","pot"]', JSON.stringify(await presets()));
await alice.click(btn("pot"));
check("pot is the most", (await alice.inputValue("#raise")) === "70", await alice.inputValue("#raise"));
await alice.fill("#raise", "500");
check("more than the pot is clamped", (await alice.locator(btn("raise to 70")).count()) === 1);
await shot(alice, "plo-4-raise");
await alice.press("#raise", "Enter");
await bob.waitForSelector(btn("call 60"));

// Omaha 5 heads up on phones
const [dave, erin] = await seat(browser, "plo5" + Date.now(), [{ name: "dave", context: phone }, { name: "erin", context: small }], game("Omaha 5", "pot limit"));
await dave.click(btn("start"));
await dave.waitForSelector(btn("call 10"));
check("five cards in Omaha 5", (await faceUp(dave)) === 5, String(await faceUp(dave)));
await shot(dave, "plo5-1-phone");
await shot(erin, "plo5-2-small");

// short deck hold'em with the J-6 bounty, heads up
const shortDeck = async (p) => {
    await p.click('fieldset:has(input[name="deck"]) >> text="short (6+)"');
    check("the bounty follows the deck", (await p.locator('legend:text-is("J-6 bounty")').count()) === 1);
    await p.click('fieldset:has(input[name="sevenDeuce"]) >> text="1 bb"');
};
const [frank, gina] = await seat(browser, "six" + Date.now(), ["frank", "gina"], shortDeck);
const rec = record(gina);
await gina.reload();
await frank.click(btn("start"));
await frank.waitForSelector(btn("call 10"));
check("the header names the game", (await frank.locator("text=6+ NLH 10/20").count()) === 1);
await checkDown([frank, gina], async () => rec.hands.length >= 3, 30000);
const seen = rec.hands.flatMap((h) => [...h.record.board, ...h.record.players.flatMap((p) => p.cards ?? [])]);
check("no card below a six", seen.length > 0 && seen.every((c) => !"2345".includes(c[0])), seen.join(" "));
check("history says 6+", rec.hands.every((h) => h.record.game === "6+ NLH"));
await shot(frank, "short-1-desktop");

await browser.close();
finish();
