// Settings tab: everyone at the table can read how the host set it up.
import { URL, check, finish, launch, shot } from "./lib.mjs";
const table = "settings" + Date.now();
const browser = await launch();
const row = (p, label) => p.locator(`[aria-label="table settings"] div:has(dt:text-is("${label}")) dd`).innerText();
const openSettings = async (p) => {
    await p.click('header button:has-text("table")');
    await p.click('[aria-label="table"] button:text-is("settings")');
};

const host = await (await browser.newContext()).newPage();
await host.goto(URL);
await host.fill("#username", "host");
await host.fill("#tablename", table);
await host.click('button:text-is("create")');
await host.click('label:has-text("custom")');
await host.fill("#sb", "25");
await host.fill("#bb", "50");
await host.fill("#minBuy", "20");
await host.fill("#maxBuy", "100");
await host.fill("#tablePassword", "s3cret");
await host.click("text=more settings");
await host.click('fieldset:has-text("turn timer") label:has-text("off")');
await host.click('button:text-is("create table")');
await host.waitForSelector("text=Open");

// a phone-sized player joins through the invite link
const p = await (await browser.newContext({ viewport: { width: 390, height: 844 } })).newPage();
await p.goto(`${URL}/?table=${table}#pw=s3cret`);
await p.fill("#username", "dan");
await p.press("#password", "Enter");
await p.waitForSelector("text=Open");
check("header shows the stakes", (await p.locator("header").innerText()).includes("25/50"));

for (const [who, page] of [["host", host], ["a player", p]]) {
    await openSettings(page);
    await page.waitForSelector('[aria-label="table settings"]');
    check(`${who} sees the stakes`, (await row(page, "stakes")) === "25/50");
    check(`${who} sees the buy-in`, (await row(page, "buy-in")) === "1,000–5,000 (20–100 bb)", await row(page, "buy-in"));
    check(`${who} sees the timer is off`, (await row(page, "turn timer")) === "off");
    check(`${who} sees the password is on`, (await row(page, "password")) === "on");
    check(`${who} sees the host`, (await row(page, "host")) === "host");
}
check("the password itself isn't shown", !(await p.locator('[aria-label="table"]').innerText()).includes("s3cret"));
await shot(host, "settings");
await shot(p, "settings-phone");
await browser.close();
finish();
