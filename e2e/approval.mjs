// Rebuy approval: rebuys and sitting back down wait for the host to approve or deny.
import { URL, check, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const gone = (p, sel) => p.waitForSelector(sel, { state: "detached", timeout: 3000 }).then(() => true).catch(() => false);
const menu = async (p, item) => {
    await p.click('button[title="seat options"]');
    await p.click(`button:text-is("${item}")`);
};
const sit = async (p, seat) => {
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "approve" + Date.now();
await alice.goto(URL);
await alice.fill("#username", "alice");
await alice.fill("#tablename", table);
await alice.click('button:text-is("create")');
await alice.click('summary:text-is("more settings")');
await alice.check("#approveRebuys");
await alice.click('button:text-is("create table")');
await alice.waitForSelector("text=Open");
await sit(alice, 1);
await bob.goto(URL);
await bob.fill("#username", "bob");
await bob.fill("#tablename", table);
await bob.click('button:text-is("join")');
await bob.waitForSelector("text=Open");
await sit(bob, 2);
check("a first buy-in needs no approval", await has(alice, 'p:text-is("bob")'));

const requests = 'ul[aria-label="buy-in requests"]';
await menu(bob, "rebuy");
await bob.fill("#buyIn", "500");
await bob.press("#buyIn", "Enter");
check("the host sees the request", await has(alice, `${requests} >> text=wants to rebuy for 500`));
check("the player sees it's waiting", await has(bob, "text=waiting for the host to ok your buy-in of 500"));
check("others can't approve", (await bob.locator(requests).count()) === 0);
await shot(alice, "approval-request");
await alice.click(`${requests} button:text-is("deny")`);
check("deny clears the request", await gone(bob, "text=waiting for the host"));
check("a denied rebuy adds nothing", !(await has(alice, "text=2500", 500)));

await menu(bob, "rebuy");
await bob.fill("#buyIn", "500");
await bob.press("#buyIn", "Enter");
await alice.click(`${requests} button:text-is("approve")`);
check("an approved rebuy lands", await has(bob, "text=2500"));

// leaving and sitting back down counts as a rebuy
await menu(bob, "leave seat");
await bob.waitForSelector("text=Open");
await sit(bob, 3);
check("sitting back down asks the host", await has(alice, `${requests} >> text=wants to sit back down`));
check("and doesn't seat them yet", !(await has(alice, 'p:text-is("bob")', 500)));
await alice.click(`${requests} button:text-is("approve")`);
check("approving seats them", await has(alice, 'p:text-is("bob")'));
await browser.close();
finish();
