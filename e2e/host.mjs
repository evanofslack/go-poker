// Host controls in the players panel: chips, banker, sit out, move to watching, remove, and passing host on.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const row = (p, name) =>
    p.locator('ul[aria-label="players"] li').filter({ has: p.locator(`span.font-semibold:text-is("${name}")`) }).innerText().catch(() => "");
const rowHas = async (p, name, text, ms = 3000) => {
    const end = Date.now() + ms;
    while (Date.now() < end) {
        if ((await row(p, name)).includes(text)) return true;
        await p.waitForTimeout(100);
    }
    return false;
};
const openPlayers = async (p) => {
    await p.click('button[aria-label^="chat"]');
    await p.click('button:text-is("players")');
};
const manage = async (p, name, item) => {
    await p.click(`button[aria-label="manage ${name}"]`);
    if (item) await p.click(`[role=dialog] button:text-is("${item}")`);
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const carol = await (await browser.newContext()).newPage();
const table = "host" + Date.now();
for (const [p, name, btn, seat] of [[alice, "alice", "create", 1], [bob, "bob", "join", 2], [carol, "carol", "join", 3]]) {
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
await openPlayers(alice);
await openPlayers(bob);
check("the host is tagged", await rowHas(alice, "alice", "host"));
check("the host can manage every row", (await alice.locator('button[aria-label^="manage "]').count()) === 3);
check("others can't manage anyone", (await bob.locator('button[aria-label^="manage "]').count()) === 0);

// chips go through the ledger
await manage(alice, "bob");
await alice.fill("#chipAmount", "500");
await alice.fill("#chipReason", "miscount");
await shot(alice, "host-manage");
await alice.click('[role=dialog] button:text-is("add chips")');
check("added chips count as bought in", await rowHas(alice, "bob", "stack 2500 · in 2500"));
await manage(alice, "bob");
await alice.click('[role=dialog] label:has-text("take")');
await alice.fill("#chipAmount", "300");
await alice.click('[role=dialog] button:text-is("take chips")');
check("taken chips count as cashed out", await rowHas(alice, "bob", "stack 2200 · in 2500 · out 300"));
check("still balanced", await has(alice, "text=· balanced"));

await manage(alice, "bob", "make banker");
check("the host can pick the banker", await rowHas(bob, "bob", "banker") && (await rowHas(bob, "alice", "even")));

await manage(alice, "carol", "sit out");
check("the host can sit a player out", await rowHas(alice, "carol", "sitting out"));
await manage(alice, "carol", "move to watching");
check("the host can move a player to watching", await rowHas(alice, "carol", "watching") && (await row(alice, "carol")).includes("out 2000"));

// removing asks first
await manage(alice, "bob", "remove from table");
check("removing asks first", await has(alice, "[role=dialog] >> text=remove bob?"));
check("the dialog takes keyboard focus", await alice.evaluate(() => !!document.activeElement?.closest("[role=dialog]")));
await alice.keyboard.press("Escape");
check("escape cancels removing", (await alice.locator("[role=dialog]").count()) === 0 && (await rowHas(alice, "bob", "playing")));
await manage(alice, "bob", "remove from table");
await alice.click('[role=dialog] button:text-is("remove")');
check("a removed player is back at the join form", await has(bob, "#username"));
check("they're told why", await has(bob, "text=the host removed you from the table"));
check("they're listed as left", await rowHas(alice, "bob", "left"));
await bob.reload();
check("a removed player doesn't rejoin on reload", await has(bob, "#username"));

// handing the host role to carol
await manage(alice, "carol", "make host");
check("the old host can't manage anymore", await alice.waitForSelector('button[aria-label^="manage "]', { state: "detached", timeout: 3000 }).then(() => true).catch(() => false));
await openPlayers(carol);
check("the new host can", await rowHas(carol, "carol", "host") && (await carol.locator('button[aria-label^="manage "]').count()) === 3);
await browser.close();
finish();
