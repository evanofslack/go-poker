// Admin page: only on the admin address, it shows a table's people with their addresses, and
// the audit of what they did and tried.
import { URL, btn, check, create, finish, has, launch, open, shot, until } from "./lib.mjs";
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

await browser.close();
finish();
