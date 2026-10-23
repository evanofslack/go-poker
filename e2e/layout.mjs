// A full nine-seat table on desktop, tablet and phone viewports, then smaller tables on phones.
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
        // the plate, and the tag hanging off its bottom with your hand
        const el = [...document.querySelectorAll("p.truncate")].find((e) => e.textContent === name).parentElement;
        const tag = el.querySelector("[data-made-hand], [data-won]");
        const plate = { ...el.getBoundingClientRect().toJSON(), bottom: Math.max(el.getBoundingClientRect().bottom, tag?.getBoundingClientRect().bottom ?? 0) };
        const controls = [...document.querySelectorAll("footer button, footer input")]
            .map((e) => e.getBoundingClientRect())
            .filter((r) => r.left < plate.right && r.right > plate.left)
            .map((r) => r.top);
        return Math.min(Infinity, ...controls) - plate.bottom;
    }, name);

const names = ["alice", "bob", "carol", "dave", "erin", "frank", "grace", "heidi", "ivan"];
const kinds = ["desktop", "phone", "tablet", "landscape", "small", "laptop", "desktop", "desktop", "phone"];

// checkLayout checks the view of each of the n players at a table
async function checkLayout(pages, n) {
    for (const { p, kind, name } of pages) {
        const overflow = await p.evaluate(() => document.documentElement.scrollWidth > window.innerWidth || document.documentElement.scrollHeight > window.innerHeight);
        check(`${n} players, ${kind}: no page scroll`, !overflow);
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
        check(`${n} players, ${kind}: own seat at the bottom`, mine && mine.y === lowest && Math.abs(mine.x - vw / 2) < vw * 0.1, JSON.stringify(mine));
        check(`${n} players, ${kind}: all ${n} seats on screen`, boxes.length === n && boxes.every((b) => b.l >= 0 && b.r <= vw), `${boxes.length}`);
        const overlaps = await p.evaluate(() => {
            const plates = [...document.querySelectorAll("p.truncate")].map((e) => [e.textContent, e.parentElement.getBoundingClientRect()]);
            const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
            return plates.flatMap(([n, a], i) => plates.slice(i + 1).filter(([, b]) => hit(a, b)).map(([m]) => `${n}/${m}`));
        });
        check(`${n} players, ${kind}: seats don't overlap`, overlaps.length === 0, overlaps.join(", "));
        // whole seats, cards included, keep a margin from the screen edges
        const edge = await p.evaluate(() =>
            [...document.querySelectorAll("[data-seat]")].filter((e) => e.firstElementChild)
                .map((e) => [e.dataset.seat, e.getBoundingClientRect()])
                .filter(([, r]) => r.left < 4 || r.right > window.innerWidth - 4)
                .map(([id, r]) => `seat ${id} ${Math.round(r.left)}-${Math.round(r.right)}`)
        );
        check(`${n} players, ${kind}: seats clear of the screen edges`, edge.length === 0, edge.join(", "));
        // the dealer button, blinds and bets stay clear of every seat and its cards
        const covered = await p.evaluate(() => {
            const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
            const seats = [...document.querySelectorAll("[data-seat]")].filter((e) => e.firstElementChild).map((e) => [e.dataset.seat, e.firstElementChild.getBoundingClientRect()]);
            return [...document.querySelectorAll("[data-markers]")].flatMap((m) =>
                seats.filter(([, r]) => hit(m.getBoundingClientRect(), r)).map(([id]) => `${m.textContent} on seat ${id}`)
            );
        });
        check(`${n} players, ${kind}: markers clear of the seats`, covered.length === 0, covered.join(", "));
        // on a phone, no seat's cards cover another seat's name and stack
        const cardsOver = await p.evaluate(() => {
            const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
            const seats = [...document.querySelectorAll("[data-seat]")].filter((e) => e.firstElementChild);
            const plates = seats.flatMap((e) => [...e.querySelectorAll("p.truncate")].map((n) => [e.dataset.seat, n.parentElement.getBoundingClientRect()]));
            return seats.flatMap((e) =>
                [...e.querySelectorAll('[role="img"]')].flatMap((c) =>
                    plates.filter(([id, r]) => id !== e.dataset.seat && hit(c.getBoundingClientRect(), r)).map(([id]) => `seat ${e.dataset.seat} cards on seat ${id}`)
                )
            );
        });
        // on a phone, a bet sits closer to its own seat than to any other
        if (["phone", "small"].includes(kind)) {
            const boardOver = await p.evaluate(() => {
                const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
                const board = document.querySelector("[data-board]").getBoundingClientRect();
                return [...document.querySelectorAll("[data-seat]")].filter((e) => e.firstElementChild).filter((e) => hit(e.firstElementChild.getBoundingClientRect(), board)).map((e) => e.dataset.seat);
            });
            check(`${n} players, ${kind}: board clear of the seats`, boardOver.length === 0, boardOver.join(", "));
            check(`${n} players, ${kind}: cards clear of other seats`, cardsOver.length === 0, [...new Set(cardsOver)].join(", "));
            const astray = await p.evaluate(() => {
                const mid = (r) => [r.left + r.width / 2, r.top + r.height / 2];
                const seats = [...document.querySelectorAll("[data-seat]")].filter((e) => e.firstElementChild).map((e) => [e.dataset.seat, mid(e.firstElementChild.getBoundingClientRect())]);
                return [...document.querySelectorAll("[data-markers]")].flatMap((m) => {
                    const [x, y] = mid(m.getBoundingClientRect());
                    const near = seats.reduce((a, b) => (Math.hypot(b[1][0] - x, b[1][1] - y) < Math.hypot(a[1][0] - x, a[1][1] - y) ? b : a));
                    const own = m.closest("[data-seat]")?.dataset.seat;
                    return near[0] === own ? [] : [`${m.textContent} of seat ${own} nearest seat ${near[0]}`];
                });
            });
            check(`${n} players, ${kind}: bets by their own seat`, astray.length === 0, astray.join(", "));
            const badges = await p.locator("[data-seat] [data-badges]").count();
            check(`${n} players, ${kind}: dealer and blinds on the seats`, badges >= 2, `${badges}`);
        }
        const gap = await footerGap(p, name);
        check(`${n} players, ${kind}: own seat clear of the footer`, gap >= 0, `${gap}px`);
        await shot(p, `lay-1-${n}-${kind}-${name}`);
    }
}

const browser = await launch();

// seatTable makes a table of n seats, sits a player in each seat of order, on the kinds
// of screen given, and starts a hand
async function seatTable(n, kinds, order) {
    const table = `lay${n}` + String(Date.now()).slice(1);
    const pages = [];
    for (let i = 0; i < order.length; i++) {
        const p = await (await browser.newContext(views[kinds[i]])).newPage();
        await p.goto(URL);
        await p.fill("#username", names[i]);
        await p.fill("#tablename", table);
        await p.click(`button:text-is("${i === 0 ? "create" : "join"}")`);
        if (i === 0) {
            await p.click("text=more settings");
            await p.click(`fieldset:has-text("seats") label:has-text("${n}")`);
            await confirmCreate(p);
        }
        await p.waitForSelector("text=Open");
        if (order.length === 9 && i === 0) await shot(p, `lay-0-empty-desktop`);
        if (order.length === 9 && i === 1) await shot(p, `lay-0-empty-phone`);
        await p.locator("button", { hasText: "Open" }).filter({ hasText: String(order[i]) }).click();
        if (order.length === 9 && i === 1) await shot(p, `lay-0-buyin-phone`);
        await p.locator("#buyIn ~ button").first().click();
        await p.waitForTimeout(200);
        pages.push({ p, kind: kinds[i], name: names[i] });
    }
    await pages[0].p.click('button:text-is("start")');
    await pages[0].p.waitForTimeout(800);
    return pages;
}

// seats are taken out of order, so seat order isn't join order
const pages = await seatTable(9, kinds, [3, 1, 6, 8, 2, 5, 7, 4, 9]);
await checkLayout(pages, 9);


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

// the server limits connections from one address, so the nine players leave first
for (const { p } of pages) await p.context().close();
// on phones: the usual six-max table, a full eight, and four players at a nine-seat table
for (const [n, order] of [[6, [2, 5, 1, 4, 6, 3]], [8, [4, 8, 2, 6, 1, 7, 3, 5]], [9, [3, 1, 7, 5]]]) {
    const small = await seatTable(n, order.map((_, i) => (i % 2 ? "small" : "phone")), order);
    await checkLayout(small, order.length);
    // whoever is to act raises: on a phone the presets and steppers fit in the footer
    for (const { p, kind, name } of small) {
        const r = p.locator('button:text-is("raise")');
        if (!(await r.count())) continue;
        await r.click();
        const before = Number(await p.inputValue("#raise"));
        await p.click('button[aria-label="more"]');
        const after = Number(await p.inputValue("#raise"));
        check(`${n} seats, ${kind}: + adds a big blind`, after === before + 20, `${before} to ${after}`);
        check(`${n} seats, ${kind}: raise presets in the footer`, (await p.locator('footer button:text-is("pot")').count()) === 1);
        check(`${n} seats, ${kind}: raise fits`, !(await p.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)));
        const gap = await footerGap(p, name);
        check(`${n} seats, ${kind}: own seat clear of the raise controls`, gap >= 0, `${gap}px`);
        await shot(p, `lay-2-raise-${n}-${kind}`);
        await p.keyboard.press("Escape");
    }
    for (const { p } of small) await p.context().close();
}

await browser.close();
finish();
