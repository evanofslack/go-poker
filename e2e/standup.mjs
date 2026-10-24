// Stand-up game: the host starts a round, the winner shows to sit down, and the
// last one standing pays.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "su" + Date.now();
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

// alice, hosting, starts a round for 2 big blinds from the host tab
await alice.click('header button:has-text("table")');
await alice.click('[aria-label="table"] button:text-is("host")');
await alice.click('label:has(input[name="standUpPayout"]) >> text="2 bb"');
await alice.click('button:text-is("start stand-up")');
check("the round waits for the next hand", await has(alice, "text=starts with the next hand"));
await alice.click('[aria-label="close table panel"]');
check("bob gets no host controls for it", (await bob.locator('button:text-is("start stand-up")').count()) === 0);

await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10")');
check("both start standing", (await bob.locator('text="standing"').count()) === 2);
await shot(bob, "standup-standing");

// alice folds, so bob wins without a call and has to show to sit
await alice.click('button:text-is("fold"):enabled');
check("bob can show to sit down", await has(bob, 'button:text-is("show to sit down")'));
await bob.click('button:text-is("show to sit down")');
check("alice pays as the last one standing", await has(alice, "text=alice pays the stand-up: 40 to 1 player"));
check("the round is over", await bob.locator('text="standing"').first().waitFor({ state: "detached", timeout: 3000 }).then(() => true).catch(() => false));

// the host can run it again from the footer, without stopping the game
check("alice can run it again", await has(alice, 'button:text-is("stand-up again")'));
check("only the host", (await bob.locator('button:text-is("stand-up again")').count()) === 0);
await alice.click('button:text-is("stand-up again")');
await alice.click('header button:has-text("table")');
await alice.click('[aria-label="table"] button:text-is("host")');
check("a new round is set for the next hand", await has(alice, "text=stand-up round, 40 each"));
await shot(alice, "standup-host");

await browser.close();
finish();
