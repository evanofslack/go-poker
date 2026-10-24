// Join form, create table dialog and buy-in form behavior.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const browser = await launch();
const p = await (await browser.newContext()).newPage();
const table = "forms" + Date.now();
await p.goto(URL);
check("username has focus", await p.evaluate(() => document.activeElement?.id === "username"));
await p.fill("#username", "carol");
await p.fill("#tablename", table);
await p.press("#tablename", "Enter");
check("enter joins (no such table yet)", !!(await p.waitForSelector("text=/no table|not found|doesn't exist|does not exist/i", { timeout: 3000 }).catch(() => null)));

// the create dialog
await p.click('button:text-is("create")');
check("create opens the settings dialog", !!(await p.waitForSelector(`text=create table ${table}`, { timeout: 3000 }).catch(() => null)));
check("old join error is cleared", (await p.locator('[role="dialog"] >> text=/not found/i').count()) === 0);
check("buy-in shows chips", (await p.locator("text=1,000–4,000 chips").count()) === 1);
await p.click('[aria-label="about ante"]');
check("a hint opens under its label", await p.locator("text=a share of the big blind").isVisible());
await shot(p, "create-table");
await p.fill("#maxBuy", "1001");
check("over-cap max buy-in is explained", await p.locator("text=max buy-in can't be more than 1000 big blinds").isVisible());
check("over-cap max buy-in disables create", await p.locator('button:text-is("create table")').isDisabled());
await p.press("#maxBuy", "Escape");
check("escape closes the dialog", (await p.locator('[role="dialog"]').count()) === 0);
await p.click('button:text-is("create")');
check("reopened dialog is back to defaults", (await p.inputValue("#maxBuy")) === "200");
await confirmCreate(p);
await p.waitForSelector("text=Open");

// buy-in form at the default settings
await p.locator("button", { hasText: "Open" }).filter({ hasText: "3" }).click();
check("buy-in shows min and max", (await p.locator("text=1000 to 4000").count()) === 1);
await p.fill("#buyIn", "20000");
check("over-max disables sit down", await p.locator('button:text-is("sit down")').isDisabled());
await p.press("#buyIn", "Escape");
check("escape cancels buy-in", (await p.locator("#buyIn").count()) === 0);
await p.locator("button", { hasText: "Open" }).filter({ hasText: "3" }).click();
await p.fill("#buyIn", "1500");
await p.press("#buyIn", "Enter");
check("enter sits down with the amount", !!(await p.waitForSelector("text=1500", { timeout: 3000 }).catch(() => null)));

// a taken name keeps the dialog open with the server's error
const q = await (await browser.newContext()).newPage();
await q.goto(URL);
await q.fill("#username", "dave");
await q.fill("#tablename", table);
await q.click('button:text-is("create")');
await confirmCreate(q);
check("taken table name shows in the dialog", !!(await q.waitForSelector('[role="dialog"] >> text=table already exists', { timeout: 3000 }).catch(() => null)));

// custom settings carry through to the table, at phone width
const r = await (await browser.newContext({ viewport: { width: 375, height: 667 } })).newPage();
await r.goto(URL);
await r.fill("#username", "erin");
await r.fill("#tablename", table + "c");
await r.click('button:text-is("create")');
await r.click('label:has-text("custom")');
await r.fill("#sb", "25");
await r.fill("#bb", "50");
await r.fill("#minBuy", "20");
await r.fill("#maxBuy", "100");
await r.click('fieldset:has-text("seats") label:has-text("6")');
await r.click('fieldset:has-text("turn timer") label:has-text("off")');
const box = await r.locator('[role="dialog"]').boundingBox();
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
