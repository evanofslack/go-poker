// Password tables: the join form asks for the password, invite links carry it, reloads keep it.
import { URL, check, finish, launch, shot } from "./lib.mjs";
const table = "pw" + Date.now();
const browser = await launch();

// the host sets the password when creating the table
const host = await (await browser.newContext()).newPage();
await host.goto(URL);
await host.fill("#username", "host");
await host.fill("#tablename", table);
await host.click('button:text-is("create")');
await host.fill("#tablePassword", "s3cret");
await host.click('button:text-is("create table")');
await host.waitForSelector("text=Open");

const p = await (await browser.newContext()).newPage();
await p.goto(`${URL}/?table=${table}`);
check("no password field before it's needed", (await p.locator("#password").count()) === 0);
await p.fill("#username", "dan");
await p.press("#username", "Enter");
check("joining asks for the password", !!(await p.waitForSelector("#password", { timeout: 3000 }).catch(() => null)));
check("password field has focus", await p.evaluate(() => document.activeElement?.id === "password"));
await p.fill("#password", "nope");
await p.press("#password", "Enter");
check("wrong password is refused", !!(await p.waitForSelector("text=wrong password", { timeout: 3000 }).catch(() => null)));
await p.fill("#password", "s3cret");
await p.press("#password", "Enter");
check("right password joins", !!(await p.waitForSelector("text=Open", { timeout: 3000 }).catch(() => null)));
await p.click('header button:has-text("table")');
await p.click('[aria-label="table"] button:text-is("settings")');
const locked = p.locator('[aria-label="table settings"] div:has(dt:text-is("password")) dd');
check("settings show the table is locked", (await locked.innerText()) === "on");
await p.click('[aria-label="close table panel"]');
await shot(p, "password-table");
await p.reload();
check("reload rejoins without asking", !!(await p.waitForSelector("text=Open", { timeout: 5000 }).catch(() => null)));

const ctx = await browser.newContext({ permissions: ["clipboard-read", "clipboard-write"] });
const q = await ctx.newPage();
await q.goto(`${URL}/?table=${table}#pw=s3cret`);
await q.waitForSelector("#password");
check("invite link fills in the password", (await q.inputValue("#password")) === "s3cret");
check("password is taken out of the address bar", !q.url().includes("pw="), q.url());
await q.fill("#username", "erin");
await q.press("#username", "Enter");
check("invite link joins", !!(await q.waitForSelector("text=Open", { timeout: 3000 }).catch(() => null)));
check("address bar has no password after joining", !q.url().includes("pw="), q.url());
await q.click('button[title="copy invite link"]');
const copied = await q.evaluate(() => navigator.clipboard.readText());
check("copied invite carries the password", copied.endsWith(`?table=${table}#pw=s3cret`), copied);

await browser.close();
finish();
