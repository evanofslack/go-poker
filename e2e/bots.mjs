// Bots: the host adds bots of each level, plays a hand against them, removes one, and starts
// a game of only bots without taking a seat.
import { check, checkDown, create, finish, has, join, last, launch, open, panel, quick, record, shot, sit, until } from "./lib.mjs";

const names = ["harper", "eric", "yasmin", "robert", "gus", "rishi", "venetia", "kenny", "daria", "sweetpea", "henry", "jesse"];

const levels = (rec) => (last(rec)?.players ?? []).map((p) => p.bot).filter(Boolean).sort();
const playedHand = (rec) => rec.hands.length > 0;

const browser = await launch();

// a seated host adds two bots from the empty seats
const alice = await open(browser);
const aRec = record(alice);
const table = "bots" + Date.now();
await create(alice, "alice", table, quick);
await sit(alice, 1);
await alice.waitForSelector("text=+ bot");

await alice.locator("button", { hasText: "+ bot" }).filter({ hasText: "2" }).click();
const suggested = await alice.inputValue('input[maxlength="20"]');
check("bot name is suggested from the list", names.includes(suggested), suggested);
await alice.fill('input[maxlength="20"]', "harper");
await shot(alice, "bots-1-dialog");
await alice.click('button:text-is("add bot")');
check("bots default to medium, with an icon", await has(alice, 'p:has-text("harper") >> [data-bot-badge][title="medium bot"]'));

await alice.locator("button", { hasText: "+ bot" }).filter({ hasText: "3" }).click();
const second = await alice.inputValue('input[maxlength="20"]');
check("suggested name skips taken ones", second !== "harper", second);
await alice.click('label:has-text("easy")');
await alice.click('button:text-is("add bot")');
await until(() => last(aRec)?.players.length === 3);
check("server sends each bot's level", JSON.stringify(levels(aRec)) === '["easy","medium"]', JSON.stringify(levels(aRec)));
check("easy bot shows its level on hover", await alice.locator('[data-seat] [title="easy bot"]').count() === 1);
check("seats don't spell out bot levels", await alice.locator("[data-seat] >> text=/(easy|medium|hard) bot/").count() === 0);

// the host's remove buttons sit clear of the badges
const clash = await alice.evaluate(() => {
    const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
    const badges = [...document.querySelectorAll("[data-bot-badge]")].map((e) => e.getBoundingClientRect());
    return [...document.querySelectorAll('button[title="remove bot"]')].some((x) => badges.some((b) => hit(x.getBoundingClientRect(), b)));
});
check("remove buttons don't cover the badges", !clash);

// on a phone the icon is whole, with the level on hover
const phone = await open(browser, { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
await join(phone, "pat", table);
await phone.waitForSelector("[data-bot-badge]");
const badges = await phone.evaluate(() =>
    [...document.querySelectorAll("[data-bot-badge]")].map((e) => ({
        text: e.innerText.trim(),
        title: e.title,
        whole: e.getBoundingClientRect().right <= e.parentElement.getBoundingClientRect().right + 1,
    }))
);
check(
    "phone bot icons are whole",
    badges.length === 2 && badges.every((b) => b.text === "" && b.whole && / bot$/.test(b.title)),
    JSON.stringify(badges)
);
await shot(phone, "bots-phone");
await phone.context().close();

// the players panel badges bots, with their buy-in in the ledger
await panel(alice, "players");
const harperRow = await alice.locator('ul[aria-label="players"] li').filter({ hasText: "harper" }).innerText();
check("players panel badges bots, with their buy-in", harperRow.includes("medium bot") && /in \d+/.test(harperRow), harperRow);
await alice.click('button[aria-label="manage harper"]');
const offered = await alice.locator("[role=dialog] button").allInnerTexts();
check("bots can't be made host, banker or a watcher", !offered.some((t) => /make host|make banker|move to watching/.test(t)), offered.join(", "));
await alice.keyboard.press("Escape");
await alice.click('button[aria-label="close table panel"]');

// alice checks or calls on her turns until a hand finishes
await alice.click('button:text-is("start")');
check("a hand against bots finishes", await checkDown([alice], () => playedHand(aRec), 30000));
check(
    "bots act on their own",
    aRec.hands.some((h) => h.record.actions.some((a) => a.name === "harper" && !a.kind.endsWith("blind"))),
    JSON.stringify(aRec.hands.at(-1)?.record.actions),
);
await shot(alice, "bots-2-hand");

await alice.click('button[aria-label="remove harper"]');
check("host removes a bot", await until(() => !last(aRec)?.players.some((p) => p.name === "harper"), 10000));

// a host without a seat runs a game of only bots
const bob = await open(browser);
const bRec = record(bob);
await create(bob, "bob", "botsonly" + Date.now(), quick);
for (const seat of ["1", "2"]) {
    await bob.locator("button", { hasText: "Open" }).filter({ hasText: seat }).click();
    await bob.click('button:text-is("add a bot here instead")');
    await bob.click('button:text-is("add bot")');
    await until(() => last(bRec)?.players.length === Number(seat));
}
await bob.click('button:text-is("start")');
// at the default BOT_DELAY bots think 1-2s a move, so two hands can take most of a minute
check("bots-only game plays hands", await until(() => bRec.hands.length >= 2, 90000));
check("spectating host sees no bot cards mid-hand", bRec.views.filter((g) => g.betting && g.street !== "preflop").every((g) => g.players.every((p) => p.cards[0] === 0) || g.players.every((p) => !p.in || p.stack === 0)));
await shot(bob, "bots-3-alone");

check("no errors", aRec.errors.length === 0 && bRec.errors.length === 0, JSON.stringify([...aRec.errors, ...bRec.errors]));
await browser.close();
finish();
