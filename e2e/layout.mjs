// A full nine-seat table on desktop, tablet and phone viewports.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";

const views = {
    desktop: { viewport: { width: 1280, height: 720 } },
    laptop: { viewport: { width: 1440, height: 900 } },
    tablet: { viewport: { width: 768, height: 1024 }, isMobile: true, hasTouch: true },
    phone: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
    small: { viewport: { width: 360, height: 740 }, isMobile: true, hasTouch: true },
    landscape: { viewport: { width: 844, height: 390 }, isMobile: true, hasTouch: true },
};
// gap between the bottom of name's seat plate and the footer's controls below it, negative on
// overlap. Controls off to the side of the plate, like the raise presets on a wide screen, don't count.
const footerGap = (p, name) =>
    p.evaluate((name) => {
        const plate = [...document.querySelectorAll("p.truncate")].find((e) => e.textContent === name).parentElement.getBoundingClientRect();
        const controls = [...document.querySelectorAll("footer button, footer input")]
            .map((e) => e.getBoundingClientRect())
            .filter((r) => r.left < plate.right && r.right > plate.left)
            .map((r) => r.top);
        return Math.min(Infinity, ...controls) - plate.bottom;
    }, name);

const names = ["alice", "bob", "carol", "dave", "erin", "frank", "grace", "heidi", "ivan"];
const kinds = ["desktop", "phone", "tablet", "landscape", "small", "laptop", "desktop", "desktop", "phone"];

const browser = await launch();
const table = "lay" + Date.now();
const pages = [];
for (let i = 0; i < 9; i++) {
    const p = await (await browser.newContext(views[kinds[i]])).newPage();
    await p.goto(URL);
    await p.fill("#username", names[i]);
    await p.fill("#tablename", table);
    await p.click(`button:text-is("${i === 0 ? "create" : "join"}")`);
    if (i === 0) {
        await p.click("text=more settings");
        await p.click('fieldset:has-text("seats") label:has-text("9")');
        await confirmCreate(p);
    }
    await p.waitForSelector("text=Open");
    if (i === 0) await shot(p, `lay-0-empty-desktop`);
    if (i === 1) await shot(p, `lay-0-empty-phone`);
    // seats are taken out of order, so seat order isn't join order
    const seat = [3, 1, 6, 8, 2, 5, 7, 4, 9][i];
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    if (i === 1) await shot(p, `lay-0-buyin-phone`);
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(200);
    pages.push({ p, kind: kinds[i], name: names[i] });
}
await pages[0].p.click('button:text-is("start")');
await pages[0].p.waitForTimeout(800);

for (const { p, kind, name } of pages) {
    const overflow = await p.evaluate(() => document.documentElement.scrollWidth > window.innerWidth || document.documentElement.scrollHeight > window.innerHeight);
    check(`${kind}: no page scroll`, !overflow);
    // own seat sits lowest on the screen, near the middle
    const boxes = await p.evaluate(() =>
        [...document.querySelectorAll("p.truncate")].map((e) => {
            const r = e.getBoundingClientRect();
            return { name: e.textContent, x: r.x + r.width / 2, y: r.y, l: r.left, r: r.right };
        })
    );
    const mine = boxes.find((b) => b.name === name);
    const lowest = Math.max(...boxes.map((b) => b.y));
    const vw = await p.evaluate(() => window.innerWidth);
    check(`${kind}: own seat at the bottom`, mine && mine.y === lowest && Math.abs(mine.x - vw / 2) < vw * 0.1, JSON.stringify(mine));
    check(`${kind}: all 9 seats on screen`, boxes.length === 9 && boxes.every((b) => b.l >= 0 && b.r <= vw), `${boxes.length}`);
    const overlaps = await p.evaluate(() => {
        const plates = [...document.querySelectorAll("p.truncate")].map((e) => [e.textContent, e.parentElement.getBoundingClientRect()]);
        const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
        return plates.flatMap(([n, a], i) => plates.slice(i + 1).filter(([, b]) => hit(a, b)).map(([m]) => `${n}/${m}`));
    });
    check(`${kind}: seats don't overlap`, overlaps.length === 0, overlaps.join(", "));
    // whole seats, cards included, keep a margin from the screen edges
    const edge = await p.evaluate(() =>
        [...document.querySelectorAll("[data-seat]")]
            .map((e) => [e.dataset.seat, e.getBoundingClientRect()])
            .filter(([, r]) => r.left < 4 || r.right > window.innerWidth - 4)
            .map(([id, r]) => `seat ${id} ${Math.round(r.left)}-${Math.round(r.right)}`)
    );
    check(`${kind}: seats clear of the screen edges`, edge.length === 0, edge.join(", "));
    // the dealer button, blinds and bets stay clear of every seat and its cards
    const covered = await p.evaluate(() => {
        const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
        const seats = [...document.querySelectorAll("[data-seat]")].map((e) => [e.dataset.seat, e.firstElementChild.getBoundingClientRect()]);
        return [...document.querySelectorAll("[data-markers]")].flatMap((m) =>
            seats.filter(([, r]) => hit(m.getBoundingClientRect(), r)).map(([id]) => `${m.textContent} on seat ${id}`)
        );
    });
    check(`${kind}: markers clear of the seats`, covered.length === 0, covered.join(", "));
    const gap = await footerGap(p, name);
    check(`${kind}: own seat clear of the footer`, gap >= 0, `${gap}px`);
    await shot(p, `lay-1-${kind}-${name}`);
}

// open the raise input on whoever is to act, and the chat on a phone
for (const { p, kind, name } of pages) {
    const r = p.locator('button:text-is("raise")');
    if (await r.count()) {
        await r.click();
        await shot(p, `lay-2-raise-${kind}`);
        check(`${kind}: raise input fits`, !(await p.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)));
        const gap = await footerGap(p, name);
        check(`${kind}: own seat clear of the raise input`, gap >= 0, `${gap}px`);
    }
}

// the seat menu opens upward from the bottom seat, so it stays on screen
for (const { p, kind } of pages.filter((x) => ["phone", "landscape"].includes(x.kind))) {
    await p.click('button[title="seat options"]');
    const menu = await p.locator('button:text-is("leave seat")').boundingBox();
    const vh = await p.evaluate(() => window.innerHeight);
    check(`${kind}: seat menu on screen`, menu && menu.y >= 0 && menu.y + menu.height <= vh, JSON.stringify(menu));
    await shot(p, `lay-4-menu-${kind}`);
    await p.keyboard.press("Escape");
    check(`${kind}: escape closes the seat menu`, (await p.locator('button:text-is("leave seat")').count()) === 0);
}
const phone = pages[1].p;
await phone.click('button:text-is("chat")');
await phone.fill('input[placeholder="say something..."]', "hi from the phone").catch(() => {});
await shot(phone, `lay-3-chat-phone`);

await browser.close();
finish();
