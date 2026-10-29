// Stats tab: net over time and per-player totals, built from a snapshot per hand.
import { btn, check, finish, fold, has, join, launch, open, panel, quick, seat, shot } from "./lib.mjs";

const browser = await launch();
const table = "stats" + Date.now();
const [alice, bob] = await seat(browser, table, ["alice", "bob"], quick);
await panel(alice, "stats");
check("no stats before a hand", await has(alice, "text=stats start after the first hand"));

await alice.click(btn("start"));
for (let i = 1; i <= 3; i++) {
    await fold([alice, bob]);
    await alice.waitForSelector(`text=${i === 1 ? "1 hand" : `${i} hands`}`, { timeout: 15000 });
}
check("counts the hands", await has(alice, "text=3 hands"));
const rows = alice.locator('table[aria-label="player stats"] tbody tr');
check("a row per player", (await rows.count()) === 2);
const aliceRow = await rows.filter({ hasText: "alice" }).innerText();
check("hands played", /^alice(\s+you)?\s+3\s/.test(aliceRow), aliceRow);
check("a signed net", /\s[+-]\d+\s+\d+$/.test(aliceRow.trim()), aliceRow);
check("a line per player", (await alice.locator('svg[aria-label="net over time per player"] path').count()) === 2);
const style = alice.locator('table[aria-label="play style"] tbody tr');
check("a play style row per player", (await style.count()) === 2);
const bobStyle = await style.filter({ hasText: "bob" }).innerText();
check("few hands show play style, marked", /%/.test(bobStyle) && (await style.filter({ hasText: "bob" }).locator('[aria-label="few hands"]').count()) === 1, bobStyle);
await alice.locator('table[aria-label="play style"] th span:text-is("VPIP")').hover();
check("columns explain themselves", await has(alice, '[role=tooltip]:has-text("voluntarily put money in")'));
await alice.mouse.move(0, 0);
await alice.waitForSelector("[role=tooltip]", { state: "detached" });
check("short tables label the lines", (await alice.locator('svg text:text-is("bob")').count()) === 1);

const chart = alice.locator('svg[aria-label="net over time per player"] rect');
const box = await chart.boundingBox();
await alice.mouse.move(box.x + box.width - 2, box.y + box.height / 2);
check("hover shows each stack", await has(alice, "[role=tooltip] >> text=hand 3"));
await shot(alice, "stats");
await alice.mouse.move(0, 0);
check("leaving the chart hides it", (await alice.locator("[role=tooltip]").count()) === 0);

// a late joiner and a reload both get the hands so far
const carol = await open(browser);
await join(carol, "carol", table);
await panel(carol, "stats");
check("a late joiner sees every hand", await has(carol, "text=3 hands"));
await bob.reload();
await bob.waitForSelector('button[aria-label^="chat"]');
await panel(bob, "stats");
check("so does a reload", await has(bob, "text=3 hands"));
await browser.close();
finish();
