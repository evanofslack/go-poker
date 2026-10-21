// Stats tab: chips over time and per-player totals, built from a snapshot per hand.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const openStats = async (p) => {
    await p.click('header button:has-text("table")');
    await p.click('[aria-label="table"] button:text-is("stats")');
};
// fold whoever is to act, ending the hand. The button can go as the turn
// passes, so try again until a click lands.
const foldHand = async (pages) => {
    const fold = 'button:text-is("fold"):enabled';
    const end = Date.now() + 15000;
    while (Date.now() < end) {
        for (const p of pages) {
            if (await p.click(fold, { timeout: 300 }).then(() => true).catch(() => false)) return;
        }
    }
    throw new Error("nobody could fold");
};

const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const table = "stats" + Date.now();
for (const [p, name, btn, seat] of [[alice, "alice", "create", 1], [bob, "bob", "join", 2]]) {
    await p.goto(URL);
    await p.fill("#username", name);
    await p.fill("#tablename", table);
    await p.click(`button:text-is("${btn}")`);
    if (btn === "create") await confirmCreate(p);
    await p.waitForSelector("text=Open");
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(300);
}
await openStats(alice);
check("no stats before a hand", await has(alice, "text=stats start after the first hand"));

await alice.click('button:text-is("start")');
for (let i = 0; i < 3; i++) {
    await foldHand([alice, bob]);
    await alice.waitForSelector(`text=${i === 0 ? "1 hand" : `${i + 1} hands`}`, { timeout: 15000 });
}
check("counts the hands", await has(alice, "text=3 hands"));
const rows = alice.locator('table[aria-label="player stats"] tbody tr');
check("a row per player", (await rows.count()) === 2);
const aliceRow = await rows.filter({ hasText: "alice" }).innerText();
check("hands played", /alice\s+3\s/.test(aliceRow), aliceRow);
check("a line per player", (await alice.locator('svg[aria-label="chips over time per player"] path').count()) === 2);
const style = alice.locator('table[aria-label="play style"] tbody tr');
check("a play style row per player", (await style.count()) === 2);
const bobStyle = await style.filter({ hasText: "bob" }).innerText();
check("folded hands show no VPIP", /^bob\s+0%\s+0%/.test(bobStyle), bobStyle);
check("few hands are grayed out", (await style.first().getAttribute("class"))?.includes("text-subtle"));
check("columns explain themselves", (await alice.locator('abbr[title^="voluntarily put money in"]').count()) === 1);
check("short tables label the lines", (await alice.locator('svg text:text-is("bob")').count()) === 1);

const chart = alice.locator('svg[aria-label="chips over time per player"] rect');
const box = await chart.boundingBox();
await alice.mouse.move(box.x + box.width - 2, box.y + box.height / 2);
check("hover shows each stack", await has(alice, '[role=tooltip] >> text=hand 3'));
await shot(alice, "stats");
await alice.mouse.move(0, 0);
check("leaving the chart hides it", (await alice.locator("[role=tooltip]").count()) === 0);

// a late joiner and a reload both get the hands so far
const carol = await (await browser.newContext()).newPage();
await carol.goto(URL);
await carol.fill("#username", "carol");
await carol.fill("#tablename", table);
await carol.click('button:text-is("join")');
await carol.waitForSelector("text=Open");
await openStats(carol);
check("a late joiner sees every hand", await has(carol, "text=3 hands"));
await bob.reload();
await bob.waitForSelector('button[aria-label^="chat"]');
await openStats(bob);
check("so does a reload", await has(bob, "text=3 hands"));
await browser.close();
finish();
