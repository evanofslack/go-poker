// Password tables: the join form asks for the password, invite links carry it, reloads keep it.
import { URL, check, create, finish, has, launch, open, panel, shot } from "./lib.mjs";
const table = "pw" + Date.now();
const browser = await launch();

// the host sets the password when creating the table
await create(await open(browser), "host", table, (p) => p.fill("#tablePassword", "s3cret"));

const p = await open(browser);
await p.goto(`${URL}/?table=${table}`);
check("no password field before it's needed", (await p.locator("#password").count()) === 0);
await p.fill("#username", "dan");
await p.press("#username", "Enter");
check("joining asks for the password", await has(p, "#password"));
check("password field has focus", await p.evaluate(() => document.activeElement?.id === "password"));
await p.fill("#password", "nope");
await p.press("#password", "Enter");
check("wrong password is refused", await has(p, "text=wrong password"));
await p.fill("#password", "s3cret");
await p.press("#password", "Enter");
check("right password joins", await has(p, "text=Open"));
await panel(p, "settings");
const locked = p.locator('[aria-label="table settings"] div:has(dt:text-is("password")) dd');
check("settings show the table is locked", (await locked.innerText()) === "on");
await p.click('[aria-label="close table panel"]');
await shot(p, "password-table");
await p.reload();
check("reload rejoins without asking", await has(p, "text=Open", 5000));

const q = await open(browser, { permissions: ["clipboard-read", "clipboard-write"] });
await q.goto(`${URL}/?table=${table}#pw=s3cret`);
await q.waitForSelector("#password");
check("invite link fills in the password", (await q.inputValue("#password")) === "s3cret");
check("password is taken out of the address bar", !q.url().includes("pw="), q.url());
await q.fill("#username", "erin");
await q.press("#username", "Enter");
check("invite link joins", await has(q, "text=Open"));
check("address bar has no password after joining", !q.url().includes("pw="), q.url());
await q.click('button[title="copy invite link"]');
const copied = await q.evaluate(() => navigator.clipboard.readText());
check("copied invite carries the password", copied.endsWith(`?table=${table}#pw=s3cret`), copied);

await browser.close();
finish();
