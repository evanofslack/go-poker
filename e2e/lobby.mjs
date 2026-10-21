// Public tables on the home page: which are listed, their order, the lock, and joining from the list.
// No table name contains "open", which other scripts match to find empty seats.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const browser = await launch();
const id = Date.now();
const [full, free, locked, hidden] = ["full", "free", "locked", "hidden"].map((s) => `lob${s}${id}`);

const page = async () => {
    const p = await (await browser.newContext()).newPage();
    await p.goto(URL);
    return p;
};

// create opens the settings dialog, change adjusts it, and the table shows once made
const create = async (table, name, change = async () => {}) => {
    const p = await page();
    await p.fill("#username", name);
    await p.fill("#tablename", table);
    await p.click('button:text-is("create")');
    await p.click('summary:text-is("more settings")');
    await change(p);
    await confirmCreate(p);
    await p.waitForSelector("text=Open");
    return p;
};

const sit = async (p) => {
    await p.locator("button", { hasText: "Open" }).first().click();
    await p.press("#buyIn", "Enter");
    await p.waitForSelector("#buyIn", { state: "detached", timeout: 3000 });
};

const row = (p, table) => p.locator(`[data-table="${table}"]`);
const rows = (p) => p.locator("[data-table]").evaluateAll((els) => els.map((e) => e.dataset.table));

// the create dialog lists new tables by default
const first = await page();
await first.fill("#username", "ann");
await first.fill("#tablename", full);
await first.click('button:text-is("create")');
await first.click('summary:text-is("more settings")');
check("public is on by default", await first.locator('label:has-text("listed on the home page") input').isChecked());
await first.click('fieldset:has-text("seats") label:has-text("2")');
await confirmCreate(first);
await first.waitForSelector("text=Open");
await sit(first);

const second = await create(free, "cat");
await sit(second);
await create(locked, "dan", (p) => p.fill("#tablePassword", "s3cret"));
await create(hidden, "eve", (p) => p.locator('label:has-text("listed on the home page") input').uncheck());

// a second player fills the heads-up table by picking it from the list
const bob = await page();
await bob.fill("#username", "bob");
await row(bob, full).waitFor({ timeout: 7000 });
await row(bob, full).click();
check("picking a table with a name joins it", !!(await bob.waitForSelector("text=Open", { timeout: 3000 }).catch(() => null)));
await sit(bob);

// a visitor sees the listed tables, refreshed by the poll
const v = await page();
const listed = async () => {
    const names = await rows(v);
    return names.includes(free) && names.includes(locked) && names.includes(full) && (await row(v, full).textContent()).includes("full");
};
const deadline = Date.now() + 8000;
while (!(await listed()) && Date.now() < deadline) await v.waitForTimeout(250);
check("public tables are listed", await listed());
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
check("picking a locked table asks for the password", !!(await v.waitForSelector("#password", { timeout: 3000 }).catch(() => null)));
check("it fills in the table", (await v.inputValue("#tablename")) === locked);
await v.fill("#password", "s3cret");
await v.press("#password", "Enter");
check("the password then joins", !!(await v.waitForSelector("text=Open", { timeout: 3000 }).catch(() => null)));

await browser.close();
finish();
