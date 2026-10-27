// Hands tab: the hands played, with only your own and shown cards, and mucked hands marked.
import { check, checkDown, finish, has, join, launch, open, panel, quick, seat, shot } from "./lib.mjs";

const hand = (p, n) => p.locator('[aria-label="hand history"] > li').filter({ has: p.locator(`button[aria-label="hand ${n}"]`) });
// cards shown in a hand's player list, once it's open
const cardsIn = async (p, n) => {
    const li = hand(p, n);
    if ((await li.locator("button[aria-expanded=false]").count()) > 0) await li.locator("button").first().click();
    return li.locator('ul[aria-label="players"] [role=img]').count();
};

const browser = await launch();
const table = "hist" + Date.now();
const [alice, bob] = await seat(browser, table, ["alice", "bob"], quick);
await panel(alice, "hands");
check("no hands before the first", await has(alice, "text=hands show up here once they're played"));

// hand 1: alice, first to act, folds
await alice.click('button:text-is("start")');
await alice.click('button:text-is("fold"):enabled', { timeout: 5000 });
check("the hand shows up", await has(alice, 'button[aria-label="hand 1"]'));

// hand 2: check or call down to showdown
await checkDown([alice, bob], async () => (await alice.locator('button[aria-label="hand 2"]').count()) > 0, 20000);
check("the showdown shows up", await has(alice, 'button[aria-label="hand 2"]'));
check("newest hand starts open", (await hand(alice, 2).locator("button").first().getAttribute("aria-expanded")) === "true");
// checked down, so the loser mucks; a split pot shows both
const mucked = await hand(alice, 2).locator("text=mucked").count();
check("the loser mucks", mucked <= 1, mucked);
check("you see your own cards and the winner's", (await cardsIn(alice, 2)) >= 2);
const pot = await hand(alice, 2).locator('ul[aria-label="pots"]').innerText();
check("pot names the hand", /^pot \d+ · .+/.test(pot.trim()), pot);
check("streets run to the river", (await hand(alice, 2).locator("p", { hasText: /^river/ }).count()) === 1);
await shot(alice, "history");

await panel(bob, "hands");
check("bob sees only his own cards from the fold", (await cardsIn(bob, 1)) === 2);
check("so does alice", (await cardsIn(alice, 1)) === 2);

// a late joiner sees shown cards only, and a reload keeps the hands
const carol = await open(browser);
await join(carol, "carol", table);
await panel(carol, "hands");
check("a late joiner gets the hands", await has(carol, "text=2 hands"));
check("with shown cards only", (await cardsIn(carol, 2)) === (mucked ? 2 : 4));
check("and none from the fold", (await cardsIn(carol, 1)) === 0);
await carol.check("text=hands I played");
check("filtering to your hands", await has(carol, "text=you haven't played a hand yet"));

await bob.reload();
await bob.waitForSelector('button[aria-label^="chat"]');
await panel(bob, "hands");
check("a reload keeps the hands", await has(bob, "text=2 hands"));
check("and your own folded-against cards", (await cardsIn(bob, 1)) === 2);
await bob.click('[aria-label="close table panel"]');
await bob.click('button[aria-label^="chat"]');
await bob.click('button:text-is("log")');
check("a reload keeps the log", await has(bob, "text=alice sits down with"));
check("without repeating lines", (await bob.locator("text=alice sits down with").count()) === 1);
check("and no poker actions", (await bob.locator("text=alice folds").count()) === 0);

await alice.setViewportSize({ width: 360, height: 740 });
await shot(alice, "history-phone");
const close = await alice.locator('[aria-label="close table panel"]').boundingBox();
check("the close button stays on a phone", close && close.x + close.width <= 360 - 8, JSON.stringify(close));
await browser.close();
finish();
