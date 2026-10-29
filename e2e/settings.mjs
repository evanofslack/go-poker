// Settings tab: everyone at the table can read how the host set it up, and the
// host can change it.
import { URL, btn, check, create, finish, fold, launch, open, panel, shot, sit, until } from "./lib.mjs";

const row = (p, label) => p.locator(`[aria-label="table settings"] dl > div:has(dt:text-is("${label}")) > dd`).innerText();
const table = "settings" + Date.now();
const browser = await launch();

const host = await open(browser, { permissions: ["clipboard-read", "clipboard-write"] });
await create(host, "host", table, async (p) => {
    await p.click('label:has-text("custom")');
    await p.fill("#sb", "25");
    await p.fill("#bb", "50");
    await p.fill("#minBuy", "20");
    await p.fill("#maxBuy", "100");
    await p.fill("#tablePassword", "s3cret");
    await p.click('fieldset:has-text("turn timer") label:has-text("off")');
});

// a phone-sized player joins through the invite link
const p = await open(browser, { viewport: { width: 390, height: 844 } });
await p.goto(`${URL}/?table=${table}#pw=s3cret`);
await p.fill("#username", "dan");
await p.press("#password", "Enter");
await p.waitForSelector("text=Open");
check("header shows the stakes", (await p.locator("header").innerText()).includes("25/50"));

for (const [who, page] of [["host", host], ["a player", p]]) {
    await panel(page, "settings");
    await page.waitForSelector('[aria-label="table settings"]');
    check(`${who} sees the stakes`, (await row(page, "stakes")) === "25/50");
    check(`${who} sees the buy-in`, (await row(page, "buy-in")) === "1,000–5,000 (20–100 bb)", await row(page, "buy-in"));
    check(`${who} sees the timer is off`, (await row(page, "turn timer")) === "off");
    check(`${who} sees the password is on`, (await row(page, "password")) === "on");
    check(`${who} sees the host`, (await page.locator('[aria-label="table settings"] p:has-text("hosted by")').innerText()) === "hosted by host");
    check(`${who} sees the sections`, (await page.locator('[aria-label="table settings"] section').count()) === 5);
}
check("shows the table name", await host.locator(`[aria-label="table settings"] >> text="${table}"`).isVisible());
await host.click('[aria-label="table settings"] button[aria-label="copy invite link"]');
const invite = await host.evaluate(() => navigator.clipboard.readText());
check("copies the invite link", invite.endsWith(`?table=${table}#pw=s3cret`), invite);
check("the password itself isn't shown", !(await p.locator('[aria-label="table"]').innerText()).includes("s3cret"));
await shot(host, "settings");
await shot(p, "settings-phone");

// the host edits the settings between hands
const edit = `[aria-label="table"] ${btn("edit")}`;
check("a player can't edit", (await p.locator(edit).count()) === 0);
await host.click(edit);
check("the form starts from the table", (await host.inputValue("#minBuy")) === "20" && (await host.inputValue("#tablePassword")) === "s3cret");
check("the seat count can't change", (await host.locator('input[name="seats"]').count()) === 0);
await shot(host, "settings-edit");
await host.click('fieldset:has(input[name="stakes"]) >> text="10/20"');
await host.fill("#tablePassword", "n3w");
await host.click(btn("save"));
check("a player sees the new stakes", await until(async () => (await row(p, "stakes")) === "10/20"));
check("buy-ins keep their big blinds", (await row(p, "buy-in")) === "400–2,000 (20–100 bb)", await row(p, "buy-in"));
await host.click('[aria-label="table settings"] button[aria-label="copy invite link"]');
const newInvite = await host.evaluate(() => navigator.clipboard.readText());
check("the invite link has the new password", newInvite.endsWith("#pw=n3w"), newInvite);

// a new player joins with it, and both sit down
const eve = await open(browser);
await eve.goto(`${URL}/?table=${table}#pw=n3w`);
await eve.fill("#username", "eve");
await eve.press("#password", "Enter");
check("the new password lets eve in", await eve.waitForSelector("text=Open").then(() => true, () => false));
await host.click('[aria-label="close table panel"]');
await sit(host, 1);
await sit(eve, 2);

// mid-hand, a change waits for the hand to end
await host.click(btn("start"));
await host.waitForSelector(btn("fold"));
await panel(host, "settings");
await panel(eve, "settings");
await host.click(edit);
await host.click('fieldset:has(input[name="stakes"]) >> text="25/50"');
check("the form says when changes apply", await host.locator("text=changes apply after this hand").first().isVisible());
await host.click(btn("save"));
check("eve sees a change is waiting", await until(() => eve.locator('text="changes apply after this hand"').isVisible()));
check("the stakes stay for this hand", (await row(eve, "stakes")) === "10/20");
await shot(eve, "settings-pending");
check("someone folds", await fold([host, eve]));
check("the change applies after the hand", await until(async () => (await row(eve, "stakes")) === "25/50"));
check("and the note goes", (await eve.locator('text="changes apply after this hand"').count()) === 0);
await browser.close();
finish();
