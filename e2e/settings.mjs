// Settings tab: everyone at the table can read how the host set it up, and the
// host can change it.
import { URL, check, finish, launch, shot } from "./lib.mjs";
const table = "settings" + Date.now();
const browser = await launch();
const row = (p, label) => p.locator(`[aria-label="table settings"] div:has(dt:text-is("${label}")) dd`).innerText();
const openSettings = async (p) => {
    await p.click('header button:has-text("table")');
    await p.click('[aria-label="table"] button:text-is("settings")');
};

const host = await (await browser.newContext({ permissions: ["clipboard-read", "clipboard-write"] })).newPage();
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
check("shows the table name", (await row(host, "table")).includes(table));
await host.click('[aria-label="table settings"] button[aria-label="copy invite link"]');
const invite = await host.evaluate(() => navigator.clipboard.readText());
check("copies the invite link", invite.endsWith(`?table=${table}#pw=s3cret`), invite);
check("the password itself isn't shown", !(await p.locator('[aria-label="table"]').innerText()).includes("s3cret"));
await shot(host, "settings");
await shot(p, "settings-phone");

// the host edits the settings between hands
const until = (fn, ms = 3000) => {
    const end = Date.now() + ms;
    const poll = async () => ((await fn()) ? true : Date.now() > end ? false : new Promise((r) => setTimeout(r, 100)).then(poll));
    return poll();
};
const edit = '[aria-label="table"] button:text-is("edit")';
check("a player can't edit", (await p.locator(edit).count()) === 0);
await host.click(edit);
check("the form starts from the table", (await host.inputValue("#minBuy")) === "20" && (await host.inputValue("#tablePassword")) === "s3cret");
check("the seat count can't change", (await host.locator('input[name="seats"]').count()) === 0);
await shot(host, "settings-edit");
await host.click('label:has(input[name="stakes"]) >> text="10/20"');
await host.fill("#tablePassword", "n3w");
await host.click('button:text-is("save")');
check("a player sees the new stakes", await until(async () => (await row(p, "stakes")) === "10/20"));
check("buy-ins keep their big blinds", (await row(p, "buy-in")) === "400–2,000 (20–100 bb)", await row(p, "buy-in"));
await host.click('[aria-label="table settings"] button[aria-label="copy invite link"]');
const newInvite = await host.evaluate(() => navigator.clipboard.readText());
check("the invite link has the new password", newInvite.endsWith("#pw=n3w"), newInvite);

// a new player joins with it, and both sit down
const eve = await (await browser.newContext()).newPage();
await eve.goto(`${URL}/?table=${table}#pw=n3w`);
await eve.fill("#username", "eve");
await eve.press("#password", "Enter");
await eve.waitForSelector("text=Open");
check("the new password lets eve in", true);
await host.click('[aria-label="close table panel"]');
for (const [page, seat] of [[host, 1], [eve, 2]]) {
    await page.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await page.locator("#buyIn ~ button").first().click();
    await page.waitForTimeout(300);
}

// mid-hand, a change waits for the hand to end
await host.click('button:text-is("start")');
await host.waitForSelector('button:text-is("fold")');
await openSettings(host);
await openSettings(eve);
await host.click(edit);
await host.click('label:has(input[name="stakes"]) >> text="25/50"');
check("the form says when changes apply", await host.locator("text=changes apply after this hand").first().isVisible());
await host.click('button:text-is("save")');
check("eve sees a change is waiting", await until(() => eve.locator('text="changes apply after this hand"').isVisible()));
check("the stakes stay for this hand", (await row(eve, "stakes")) === "10/20");
await shot(eve, "settings-pending");
const folded = await until(async () => {
    for (const page of [host, eve]) {
        if (await page.click('button:text-is("fold"):enabled', { timeout: 300 }).then(() => true).catch(() => false)) return true;
    }
    return false;
});
check("someone folds", folded);
check("the change applies after the hand", await until(async () => (await row(eve, "stakes")) === "25/50"));
check("and the note goes", (await eve.locator('text="changes apply after this hand"').count()) === 0);
await browser.close();
finish();
