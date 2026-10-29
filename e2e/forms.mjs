// Join form, create table dialog and buy-in form behavior.
import { URL, btn, check, confirmCreate, finish, gone, has, launch, open, openRules, shot } from "./lib.mjs";
const browser = await launch();
const p = await open(browser);
const table = "forms" + Date.now();
await p.goto(URL);
check("username has focus", await p.evaluate(() => document.activeElement?.id === "username"));
await p.fill("#username", "carol");
await p.fill("#tablename", table);
await p.press("#tablename", "Enter");
check("enter joins (no such table yet)", await has(p, "text=/no table|not found|doesn't exist|does not exist/i"));

// the create dialog
await p.click(btn("create"));
check("create opens the settings dialog", await has(p, `text=create table ${table}`));
check("old join error is cleared", (await p.locator('[role="dialog"] >> text=/not found/i').count()) === 0);
check("buy-in shows chips", (await p.locator("text=1,000–4,000 chips").count()) === 1);
check("pace starts closed", !(await p.locator('input[name="turnTime"]').first().isVisible()));
check("and sums itself up", await p.locator("text=30s timer · 2 time cards · 5s between hands").isVisible());
check("house rules too", await p.locator("text=run it · duck hunt").isVisible());
await p.click('[aria-label="about ante"]');
check("a hint opens", await p.locator("text=a share of the big blind").isVisible());
await shot(p, "create-table");
await p.keyboard.press("Escape");
check("escape closes the hint", await gone(p, "text=a share of the big blind"));
check("and leaves the dialog open", await p.locator('[role="dialog"]').isVisible());
await p.fill("#maxBuy", "1001");
check("over-cap max buy-in is explained", await p.locator("text=max buy-in can't be more than 1000 big blinds").isVisible());
check("over-cap max buy-in disables create", await p.locator(btn("create table")).isDisabled());
await p.press("#maxBuy", "Escape");
check("escape closes the dialog", (await p.locator('[role="dialog"]').count()) === 0);
await p.click(btn("create"));
check("reopened dialog is back to defaults", (await p.inputValue("#maxBuy")) === "200");
await confirmCreate(p);
await p.waitForSelector("text=Open");

// buy-in form at the default settings
await p.locator("button", { hasText: "Open" }).filter({ hasText: "3" }).click();
check("buy-in shows min and max", (await p.locator("text=1000 to 4000").count()) === 1);
await p.fill("#buyIn", "20000");
check("over-max disables sit down", await p.locator(btn("sit down")).isDisabled());
await p.press("#buyIn", "Escape");
check("escape cancels buy-in", (await p.locator("#buyIn").count()) === 0);
await p.locator("button", { hasText: "Open" }).filter({ hasText: "3" }).click();
await p.fill("#buyIn", "1500");
await p.press("#buyIn", "Enter");
check("enter sits down with the amount", await has(p, "text=1500"));

// a taken name keeps the dialog open with the server's error
const q = await open(browser);
await q.goto(URL);
await q.fill("#username", "dave");
await q.fill("#tablename", table);
await q.click(btn("create"));
await confirmCreate(q);
check("taken table name shows in the dialog", await has(q, '[role="dialog"] >> text=table already exists'));

// custom settings carry through to the table, at phone width
const r = await open(browser, { viewport: { width: 375, height: 667 } });
await r.goto(URL);
await r.fill("#username", "erin");
await r.fill("#tablename", table + "c");
await r.click(btn("create"));
await r.click('label:has-text("custom")');
await r.fill("#sb", "25");
await r.fill("#bb", "50");
await r.fill("#minBuy", "20");
await r.fill("#maxBuy", "100");
await r.click('fieldset:has-text("seats") label:has-text("6")');
await openRules(r);
await r.click('fieldset:has-text("turn timer") label:has-text("off")');
check("an open section sums up the change", await r.locator("text=no timer · 5s between hands").isVisible());
const box = await r.locator('[role="dialog"]').boundingBox();
const createBox = await r.locator(btn("create table")).boundingBox();
check("create stays on screen", createBox.y + createBox.height <= 667, JSON.stringify(createBox));
check("dialog fits a phone screen", box.x >= 0 && box.x + box.width <= 375, JSON.stringify(box));
await shot(r, "create-table-phone");
await confirmCreate(r);
await r.waitForSelector("text=Open");
check("table has the chosen stakes", await r.locator("text=25/50").first().isVisible());
check("table has the chosen seats", (await r.locator("button", { hasText: "Open" }).count()) === 6, String(await r.locator("button", { hasText: "Open" }).count()));
await r.locator("button", { hasText: "Open" }).first().click();
check("buy-in uses the chosen limits", (await r.locator("text=1000 to 5000").count()) === 1);
await browser.close();
finish();
