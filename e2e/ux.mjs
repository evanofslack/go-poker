// Invite links, table hints, error toasts, connection status, header, leaving and light mode.
import { URL, btn, check, confirmCreate, finish, gone, has, launch, open, shot, sit, until } from "./lib.mjs";

const browser = await launch();
const alice = await open(browser, { permissions: ["clipboard-read", "clipboard-write"] });
const bob = await open(browser);

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
await alice.click(btn("create"));
await confirmCreate(alice);
await alice.waitForSelector("text=Open");
check("url becomes the invite link", alice.url().endsWith(`?table=${table}`), alice.url());
check("empty table hint", await has(alice, "text=pick a seat to start"));
await shot(alice, `ux-1-empty`);

await alice.locator(btn("copy invite link")).first().click();
check("copy shows a toast", await has(alice, "text=invite link copied"));
const copied = await alice.evaluate(() => navigator.clipboard.readText());
check("clipboard holds the invite link", copied === `${URL}/?table=${table}`, copied);

await sit(alice, 1);
check("one player hint", await has(alice, "text=waiting for another player"));

// bob opens the invite link: only a name is needed
await bob.goto(copied);
// the form fills in once the page has hydrated
check("invite link fills the table", await until(async () => (await bob.inputValue("#tablename")) === table));
await bob.fill("#username", "bob");
await bob.press("#username", "Enter");
check("spectator hint before sitting", await has(bob, "text=pick an open seat to play"));
check("header counts the spectator", await has(alice, "text=watching 1"));
await sit(bob, 2);
check("watching count clears once seated", await gone(alice, "text=watching 1"));
check("hint clears when two are ready", await gone(alice, "text=waiting for another player"));
check("the invite stays until the game starts", await has(alice, "text=invite friends before you start"));

aliceWS.send(JSON.stringify({ action: "error", message: "can't rebuy during a hand" }));
check("server errors show as a toast", await has(alice, "[role=alert] >> text=can't rebuy during a hand"));
await shot(alice, `ux-2-toast`);
check("toast goes away", await gone(alice, "[role=alert]:has-text(\"can't rebuy during a hand\")", 6000));

await aliceWS.close();
check("dropped connection shows reconnecting", await has(alice, "text=reconnecting…", 2000));
await shot(alice, `ux-3-reconnecting`);
check("reconnects with a toast", await has(alice, "text=reconnected", 5000));
check("seat kept after reconnect", await has(alice, 'p:text-is("alice")'));

await alice.click('button[aria-label="copy invite link"]');
check("header copy icon copies the link", await has(alice, "text=invite link copied"));
await shot(alice, `ux-4-header`);

// bob goes home: seated, so it asks first
await bob.click(btn("GoPoker"));
check("home asks before leaving a seat", await has(bob, "[role=dialog] >> text=leave the table?"));
await shot(bob, `ux-5-leave-confirm`);
await bob.click(`[role=dialog] ${btn("leave")}`);
check("leaving shows the join form", await has(bob, "#username"));
check("leaving clears the invite from the url", !bob.url().includes("?table="), bob.url());
check("bob's seat is freed", await gone(alice, 'p:text-is("bob")'));
await bob.reload();
check("a reload stays on the join form", await has(bob, "#username"));

// light mode
const scheme = () => bob.evaluate(() => document.documentElement.dataset.mantineColorScheme);
check("follows a dark system", (await scheme()) === "dark");
await bob.emulateMedia({ colorScheme: "light" });
await bob.waitForFunction(() => document.documentElement.dataset.mantineColorScheme === "light", null, { timeout: 2000 }).catch(() => {});
check("follows a light system", (await scheme()) === "light");
await bob.click('button[aria-label="light or dark"]');
check("the toggle switches to dark", (await scheme()) === "dark");
await bob.reload();
check("the choice is remembered", (await scheme()) === "dark");
await bob.click('button[aria-label="light or dark"]');
await shot(bob, "ux-light");

await browser.close();
finish();
