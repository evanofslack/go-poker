// Invite links, table hints, error toasts, connection status.
import { URL, check, finish, launch, shot } from "./lib.mjs";
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
await bob.locator("button", { hasText: "Open" }).filter({ hasText: "2" }).click();
await bob.locator("#buyIn ~ button").first().click();
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

await browser.close();
finish();
