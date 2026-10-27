// Public tables on the home page: which are listed, their order, the lock, and joining from the list.
// No table name contains "open", which other scripts match to find empty seats.
import { URL, check, confirmCreate, create, finish, has, launch, open, shot, sit, until } from "./lib.mjs";
const browser = await launch();
const id = Date.now();
const [full, free, locked, hidden] = ["full", "free", "locked", "hidden"].map((s) => `lob${s}${id}`);

const page = async () => {
    const p = await open(browser);
    await p.goto(URL);
    return p;
};

// make creates table as name; change adjusts the settings dialog
const make = async (table, name, change = true) => {
    const p = await open(browser);
    await create(p, name, table, change);
    return p;
};

const row = (p, table) => p.locator(`[data-table="${table}"]`);
const rows = (p) => p.locator("[data-table]").evaluateAll((els) => els.map((e) => e.dataset.table));

// the create dialog lists new tables by default
const first = await page();
await first.fill("#username", "ann");
await first.fill("#tablename", full);
await first.click('button:text-is("create")');
check("public is on by default", await first.locator("#public").isChecked());
await first.click('fieldset:has-text("seats") label:has-text("2")');
await confirmCreate(first);
await first.waitForSelector("text=Open");
await sit(first, 1);

await sit(await make(free, "cat"), 1);
await make(locked, "dan", (p) => p.fill("#tablePassword", "s3cret"));
await make(hidden, "eve", (p) => p.locator("#public").uncheck());

// a second player fills the heads-up table by picking it from the list
const bob = await page();
await bob.fill("#username", "bob");
await row(bob, full).waitFor({ timeout: 7000 });
await row(bob, full).click();
check("picking a table with a name joins it", await has(bob, "text=Open"));
await sit(bob, 2);

// a visitor sees the listed tables, refreshed by the poll
const v = await page();
const listed = async () => {
    const names = await rows(v);
    return names.includes(free) && names.includes(locked) && names.includes(full) && (await row(v, full).textContent()).includes("full");
};
check("public tables are listed", await until(listed, 8000));
const names = await rows(v);
check("private tables aren't listed", !names.includes(hidden));
check("more players sort first", names.indexOf(free) < names.indexOf(locked), names.join(", "));
check("full tables sort last", names.indexOf(full) > names.indexOf(free) && names.indexOf(full) > names.indexOf(locked));
check("rows show stakes and seats", (await row(v, full).textContent()).includes("2/2"));
check("blinds and seats are labelled", (await v.locator('section[aria-label="public tables"] >> text=blinds').count()) === 1 && (await v.locator('section[aria-label="public tables"] >> text=seats').count()) === 1);
check("password tables show a lock", (await row(v, locked).locator('[aria-label="needs a password"]').count()) === 1);
check("open tables have no lock", (await row(v, free).locator('[aria-label="needs a password"]').count()) === 0);
await shot(v, "lobby");
await v.setViewportSize({ width: 390, height: 844 });
check("no sideways scroll on a phone", await v.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));
await shot(v, "lobby-phone");

// a locked table asks for the password instead of joining
await v.fill("#username", "vic");
await row(v, locked).click();
check("picking a locked table asks for the password", await has(v, "#password"));
check("it fills in the table", (await v.inputValue("#tablename")) === locked);
await v.fill("#password", "s3cret");
await v.press("#password", "Enter");
check("the password then joins", await has(v, "text=Open"));

await browser.close();
finish();
