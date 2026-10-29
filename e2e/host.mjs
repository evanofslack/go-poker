// Host controls in the players panel: chips, banker, sit out, move to watching, remove, and passing host on;
// and turning bomb pots on from the host tab.
import { btn, check, finish, gone, has, launch, panel, playerRow, seat, shot, until } from "./lib.mjs";

const rowHas = (p, name, text) => until(async () => (await playerRow(p, name)).includes(text));
const manage = async (p, name, item) => {
    await p.click(`button[aria-label="manage ${name}"]`);
    if (item) await p.click(`[role=dialog] ${btn(item)}`);
};

const browser = await launch();
const [alice, bob, carol] = await seat(browser, "host" + Date.now(), ["alice", "bob", "carol"]);
await panel(alice, "players");
await panel(bob, "players");
check("the host is tagged", await rowHas(alice, "alice", "host"));
check("the host can manage every row", (await alice.locator('button[aria-label^="manage "]').count()) === 3);
check("others can't manage anyone", (await bob.locator('button[aria-label^="manage "]').count()) === 0);

// chips go through the ledger
await manage(alice, "bob");
await alice.fill("#chipAmount", "500");
await alice.fill("#chipReason", "miscount");
await shot(alice, "host-manage");
await alice.click(`[role=dialog] ${btn("add chips")}`);
check("added chips count as bought in", await rowHas(alice, "bob", "stack 2500 · in 2500"));
await manage(alice, "bob");
await alice.click('[role=dialog] label:has-text("take")');
await alice.fill("#chipAmount", "300");
await alice.click(`[role=dialog] ${btn("take chips")}`);
check("taken chips count as cashed out", await rowHas(alice, "bob", "stack 2200 · in 2500 · out 300"));
check("still balanced", await has(alice, "text=· balanced"));

await manage(alice, "bob", "make banker");
check("the host can pick the banker", (await rowHas(bob, "bob", "banker")) && (await rowHas(bob, "alice", "even")));

await manage(alice, "carol", "sit out");
check("the host can sit a player out", await rowHas(alice, "carol", "sitting out"));
await manage(alice, "carol", "move to watching");
check("the host can move a player to watching", (await rowHas(alice, "carol", "watching")) && (await playerRow(alice, "carol")).includes("out 2000"));

// removing asks first
await manage(alice, "bob", "remove from table");
check("removing asks first", await has(alice, "[role=dialog] >> text=remove bob?"));
check("the dialog takes keyboard focus", await until(() => alice.evaluate(() => !!document.activeElement?.closest("[role=dialog]"))));
await alice.keyboard.press("Escape");
check("escape cancels removing", (await alice.locator("[role=dialog]").count()) === 0 && (await rowHas(alice, "bob", "playing")));
await manage(alice, "bob", "remove from table");
await alice.click(`[role=dialog] ${btn("remove")}`);
check("a removed player is back at the join form", await has(bob, "#username"));
check("they're told why", await has(bob, "text=the host removed you from the table"));
check("they're listed as left", await rowHas(alice, "bob", "left"));
await bob.reload();
check("a removed player doesn't rejoin on reload", await has(bob, "#username"));

// bomb pots are off, and the host tab goes straight to them in the settings
await panel(alice, "host");
await alice.click(btn("turn them on"));
check("turning bomb pots on opens their section", await alice.locator("#bombPots").isVisible());
await alice.click('label[for="bombPots"]');
await alice.click(`[role=dialog] ${btn("save")}`);
check("then the host tab has their controls", await has(alice, btn("deal one next hand")));
await panel(alice, "players");

// handing the host role to carol
await manage(alice, "carol", "make host");
check("the old host can't manage anymore", await gone(alice, 'button[aria-label^="manage "]'));
await panel(carol, "players");
check("the new host can", (await rowHas(carol, "carol", "host")) && (await carol.locator('button[aria-label^="manage "]').count()) === 3);
await browser.close();
finish();
