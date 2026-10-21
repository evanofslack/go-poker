// Host session controls: pause and resume, ending the session after the hand, and the final settle-up.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const hostMenu = 'button[aria-haspopup=menu]:has-text("host")';
const menu = async (p, item) => {
    await p.click(hostMenu);
    await p.click(`[role=menuitem]:text-is("${item}")`);
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "end" + Date.now();
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
check("host sees the host menu", (await alice.locator(hostMenu).count()) === 1);
check("non-host doesn't see it", (await bob.locator(hostMenu).count()) === 0);

// pausing before the start blocks it
await menu(alice, "pause");
check("everyone sees the pause", await has(bob, "text=paused by the host"));
check("start is disabled while paused", await has(alice, 'button:text-is("start"):disabled'));
await menu(alice, "resume");
check("resuming clears the hint", await bob.waitForSelector("text=paused by the host", { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("fold")');

// ending mid-hand waits for the hand
await menu(alice, "end session");
check("ending asks first", await has(alice, "[role=dialog] >> text=after this hand"));
await alice.click('[role=dialog] button:text-is("end session")');
check("everyone is told it's the last hand", await has(bob, "text=last hand"));
check("the menu drops end session once ending", await alice.click(hostMenu).then(() => alice.locator('[role=menuitem]:text-is("end session")').count()).then((n) => n === 0));
await alice.keyboard.press("Escape");
await alice.click('button:text-is("fold")');

check("everyone gets the settle-up", await has(bob, "[role=dialog] >> text=the session is over"));
const final = await bob.locator('ul[aria-label="final ledger"]').innerText().catch(() => "");
check("it shows who sends and gets what", final.includes("alice") && final.includes("-10") && final.includes("+10"), final);
await shot(bob, "session-ended");
await bob.click('[role=dialog] button:text-is("close")');
check("the table says when it closes", await has(bob, "text=table closes in 60m"));
check("open seats are gone", (await bob.locator("button", { hasText: "Open" }).count()) === 0);
check("no more start", (await alice.locator('button:text-is("start")').count()) === 0);
check("the host menu is gone", (await alice.locator(hostMenu).count()) === 0);

// a late spectator can still read it
const carol = await (await browser.newContext()).newPage();
await carol.goto(URL);
await carol.fill("#username", "carol");
await carol.fill("#tablename", table);
await carol.click('button:text-is("join")');
check("a late joiner sees the settle-up", await has(carol, "[role=dialog] >> text=the session is over"));
await browser.close();
finish();
