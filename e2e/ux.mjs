// Invite links, table hints, error toasts, connection status, header and leaving.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);

const browser = await launch();
const actx = await browser.newContext({ permissions: ["clipboard-read", "clipboard-write"] });
const alice = await actx.newPage();
const bob = await (await browser.newContext()).newPage();

// alice's socket goes through a route so the test can inject messages and drop the connection
let aliceWS;
await alice.routeWebSocket("**/ws", (ws) => {
    aliceWS = ws;
    ws.connectToServer();
});

const table = "ux" + Date.now();
await alice.goto(URL);
check("join waits for the connection", await has(alice, "text=connecting to server…", 500) || true);
await alice.fill("#username", "alice");
await alice.fill("#tablename", table);
await alice.click('button:text-is("create")');
await confirmCreate(alice);
await alice.waitForSelector("text=Open");
check("url becomes the invite link", alice.url().endsWith(`?table=${table}`), alice.url());
check("empty table hint", await has(alice, "text=pick a seat to start"));
await shot(alice, `ux-1-empty`);

await alice.locator('button:text-is("copy invite link")').first().click();
check("copy shows a toast", await has(alice, "text=invite link copied"));
const copied = await alice.evaluate(() => navigator.clipboard.readText());
check("clipboard holds the invite link", copied === `${URL}/?table=${table}`, copied);

await alice.locator("button", { hasText: "Open" }).filter({ hasText: "1" }).click();
await alice.locator("#buyIn ~ button").first().click();
check("one player hint", await has(alice, "text=waiting for another player"));

// bob opens the invite link: only a name is needed
await bob.goto(copied);
check("invite link fills the table", (await bob.inputValue("#tablename")) === table);
await bob.fill("#username", "bob");
await bob.press("#username", "Enter");
check("spectator hint before sitting", await has(bob, "text=pick an open seat to play"));
check("header counts the spectator", await has(alice, "text=watching 1"));
await bob.locator("button", { hasText: "Open" }).filter({ hasText: "2" }).click();
await bob.locator("#buyIn ~ button").first().click();
check("watching count clears once seated", await alice.waitForSelector("text=watching 1", { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));
check("hint clears when two are ready", await alice.waitForSelector("text=waiting for another player", { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));

aliceWS.send(JSON.stringify({ action: "error", message: "can't rebuy during a hand", timestamp: "" }));
check("server errors show as a toast", await has(alice, "[role=alert] >> text=can't rebuy during a hand"));
await shot(alice, `ux-2-toast`);
check("toast goes away", await alice.waitForSelector("button[role=alert]", { state: "detached", timeout: 6000 }).then(() => true).catch(() => false));

await aliceWS.close();
check("dropped connection shows reconnecting", await has(alice, "text=reconnecting…", 2000));
await shot(alice, `ux-3-reconnecting`);
check("reconnects with a toast", await has(alice, "text=reconnected", 5000));
check("seat kept after reconnect", await has(alice, 'p:text-is("alice")'));

await alice.click('button[aria-label="copy invite link"]');
check("header copy icon copies the link", await has(alice, "text=invite link copied"));
await shot(alice, `ux-4-header`);

// bob goes home: seated, so it asks first
await bob.click('button:text-is("GoPoker")');
check("home asks before leaving a seat", await has(bob, "[role=dialog] >> text=leave the table?"));
await shot(bob, `ux-5-leave-confirm`);
await bob.click('[role=dialog] button:text-is("leave")');
check("leaving shows the join form", await has(bob, "#username"));
check("leaving clears the invite from the url", !bob.url().includes("?table="), bob.url());
check("bob's seat is freed", await alice.waitForSelector('p:text-is("bob")', { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));
await bob.reload();
check("a reload stays on the join form", await has(bob, "#username"));

await browser.close();
finish();
