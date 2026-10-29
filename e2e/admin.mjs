// Admin page: only on the admin address, it shows a table's people with their addresses, the
// audit of what they did and tried, and the tables that closed.
import { URL, btn, check, create, finish, has, launch, open, shot, sit, until } from "./lib.mjs";
const ADMIN = process.env.ADMIN_URL || "http://localhost:8098";
const table = "admin" + Date.now();
const browser = await launch();

check("the public address doesn't serve it", (await fetch(`${URL}/admin`)).status === 404);

await create(await open(browser), "host", table, (p) => p.fill("#tablePassword", "s3cret"));
const intruder = await open(browser);
await intruder.goto(`${URL}/?table=${table}`);
await intruder.fill("#username", "mallory");
await intruder.press("#username", "Enter");
await intruder.fill("#password", "guess");
await intruder.press("#password", "Enter");
await intruder.waitForSelector("text=wrong password");

const p = await open(browser);
await p.goto(`${ADMIN}/admin`);
const row = `[aria-label="tables"] tr:has-text("${table}")`;
check("the table is listed", await has(p, row, 6000));
await p.click(row);
const people = `[aria-label="people"]`;
check("its people show their address", await has(p, `${people} tr:has-text("host"):has-text("127.0.0.1")`));
// the audit is written within a second
const refused = `[aria-label="audit"] tr:has-text("mallory"):has-text("wrong password")`;
const found = await until(async () => {
    await p.locator(btn("refresh")).first().click();
    return has(p, refused, 300);
}, 5000);
check("the refused join is in its audit", found);
await p.locator(refused).click();
check("a row opens to its command, without the password", await has(p, "pre:has-text('\"Name\": \"mallory\"')") &&
    !(await p.locator("pre").innerText()).includes("guess"));
await shot(p, "admin-table");

await p.locator(`${people} button:text-is("127.0.0.1")`).first().click();
check("an address looks up its audit everywhere", await has(p, `section[aria-label="lookup"] [aria-label="audit"] tr:has-text("${table}")`));
await p.locator(`section[aria-label="lookup"]`).scrollIntoViewIfNeeded();
await shot(p, "admin-lookup");

// a table whose last player leaves closes, and is kept with its final ledger
const gone = "gone" + Date.now();
const solo = await open(browser);
await create(solo, "solo", gone);
await sit(solo, 1);
await solo.click(btn("GoPoker"));
await solo.click(btn("leave"));
const closedRow = `[aria-label="closed tables"] tr:has-text("${gone}")`;
check("a closed table is listed", await has(p, closedRow, 8000));
check("with why it closed", (await p.locator(closedRow).innerText()).includes("empty"));
await p.click(closedRow);
check("it opens to its final ledger", await has(p, `[aria-label="ledger"] tr:has-text("solo"):has-text("2000")`));
check("and its log", await has(p, `text=solo left the table`));
check("and its audit", await has(p, `[aria-label="audit"] tr:has-text("solo"):has-text("take-seat")`, 5000));
await shot(p, "admin-closed");

await browser.close();
finish();
