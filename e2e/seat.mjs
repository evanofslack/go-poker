// Seat menu: sit out / in, rebuy, sit out next hand, leave seat.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const menu = async (p, item) => {
    await p.click('button[title="seat options"]');
    await p.click(`button:text-is("${item}")`);
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "seat" + Date.now();
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

check("only your own seat has a menu", (await alice.locator('button[title="seat options"]').count()) === 1);
await menu(alice, "sit out");
check("sitting out shows to others", await has(bob, "text=sitting out"));
await alice.click('button[title="seat options"]');
await shot(alice, "seat-1-menu");
await alice.click('button:text-is("sit in")');
check("sit in clears the label", await bob.waitForSelector("text=sitting out", { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));

await menu(alice, "rebuy");
check("rebuy form is capped at max minus stack", (await alice.inputValue("#buyIn")) === "2000", await alice.inputValue("#buyIn"));
await alice.fill("#buyIn", "500");
await alice.press("#buyIn", "Enter");
check("rebuy adds to the stack", await has(bob, "text=2500"));

await alice.click('button:text-is("start")');
await bob.waitForSelector("text=/^(waiting for alice|fold)$/");
await alice.click('button[title="seat options"]');
check("mid-hand: no rebuy, sit out next hand", (await alice.locator('button:text-is("rebuy")').count()) === 0 && (await alice.locator('button:text-is("sit out next hand")').count()) === 1);
await alice.click('button:text-is("sit out next hand")');
await alice.click('button[title="seat options"]');
check("server confirms sit out next hand", await has(alice, 'button:text-is("play next hand")'));
await alice.click('button[title="seat options"]');
// whoever is to act folds, ending the hand
for (const p of [alice, bob]) if (await p.locator('button:text-is("fold")').count()) await p.click('button:text-is("fold")');
check("sit-out applies when the hand ends", await has(bob, "text=sitting out", 4000));
await bob.waitForTimeout(1000);
check("no next hand dealt while alice sits out", (await bob.locator('button:text-is("fold")').count()) === 0);

await menu(bob, "leave seat");
check("leaving removes bob for others", await alice.waitForSelector('p:text-is("bob")', { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));
check("bob sees open seats mid-game", await has(bob, "text=Open"));
await shot(alice, "seat-2-after-leave");

await browser.close();
finish();
