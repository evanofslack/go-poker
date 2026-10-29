// The home page on desktop and phone, dark and light: the join form near the top, the
// features under the public tables, and their demos answering clicks.
import { URL, btn, check, finish, has, launch, open, shot } from "./lib.mjs";

const views = {
    desktop: { viewport: { width: 1280, height: 720 } },
    phone: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

// ranks of the face-up cards in a tile, like ["ace", "5"]
const ranks = (p, tile) =>
    p.locator(`${tile} [role=img][aria-label*=" of "]`).evaluateAll((els) => [
        ...new Set(els.map((e) => e.getAttribute("aria-label").split(" of ")[0])),
    ]);
const games = 'li:has([data-games])';

const browser = await launch();
for (const [kind, view] of Object.entries(views)) {
    for (const colorScheme of ["dark", "light"]) {
        const p = await open(browser, { ...view, colorScheme });
        await p.goto(URL);
        await p.waitForSelector("#username");
        const box = await p.evaluate(() => {
            const rect = (s) => document.querySelector(s).getBoundingClientRect();
            return {
                form: rect("form").bottom,
                tables: rect('[aria-label="public tables"]').bottom,
                features: rect('[aria-label="features"]').top,
                height: window.innerHeight,
                width: window.innerWidth,
                wide: document.documentElement.scrollWidth > window.innerWidth,
            };
        });
        const name = `${kind}, ${colorScheme}`;
        check(`${name}: form on the first screen`, box.form <= box.height, `bottom ${box.form}`);
        check(`${name}: features under the tables`, box.features > box.tables, `top ${box.features}`);
        check(`${name}: no sideways scroll`, !box.wide);
        const toggle = await p.locator('[aria-label="light or dark"]').boundingBox();
        check(`${name}: scheme toggle top right`, toggle.y < 60 && toggle.x > box.width / 2, JSON.stringify(toggle));
        check(`${name}: five features`, (await p.locator('[aria-label="features"] > ul > li').count()) === 5);
        await shot(p, `home-${kind}-${colorScheme}`);
        await p.locator('[aria-label="features"]').scrollIntoViewIfNeeded();
        await shot(p, `home-features-${kind}-${colorScheme}`);
        await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
        await shot(p, `home-end-${kind}-${colorScheme}`);
        await p.context().close();
    }
}

// the demos, once
const p = await open(browser, views.desktop);
await p.goto(URL);
await p.waitForSelector("[data-games]");
const count = () => p.locator(`${games} [role=img][aria-label*=" of "]`).count();
check("hold'em deals 2", (await count()) === 2);
await p.click('label[for="demo-game-omaha"]');
await p.waitForTimeout(600);
check("omaha deals 4", (await count()) === 4);
await p.click('label[for="demo-game-omaha5"]');
await p.waitForTimeout(600);
check("omaha 5 deals 5", (await count()) === 5);
await p.click('label[for="demo-deck-short"]');
await p.waitForTimeout(600);
const short = await ranks(p, games);
check("short deck deals nothing under a six", short.length > 0 && !short.some((r) => ["2", "3", "4", "5"].includes(r)), short.join(" "));
await p.click('label[for="demo-limit-pot"]');
check("pot limit names the game", await p.locator(`${games} :text("pot limit 6+ 5 card omaha")`).isVisible());

await p.click('[data-ledger] button:text-is("eric")');
const settles = await p.locator("[data-settle]").allTextContents();
check("the banker moves", settles.includes("banker") && settles.filter((s) => s.includes("eric")).length === 3, settles.join(" | "));
await p.click(`[data-ledger] li:has(button:text-is("gus")) ${btn("cash out")}`);
check("cashing out leaves", await p.locator('[data-ledger] li:has(button:text-is("gus")) :text("left")').isVisible());
check("stats list all four", (await p.locator('[aria-label="play style"] tbody tr').count()) === 4);
await p.locator('[aria-label="play style"] th span:text-is("VPIP")').hover();
check("style columns explain themselves", await has(p, '[role=tooltip]:has-text("voluntarily put money in")'));
await p.mouse.move(0, 0);

check("house rules show at once", (await p.locator('[data-rules] [role=img][aria-label*=" of "]').count()) === 12);
await p.click('[aria-label="deal the 7-2 again"]');
await p.waitForTimeout(600);
check("the 7-2 deals again face up", (await p.locator('[aria-label="deal the 7-2 again"] [aria-label="7 of clubs"]').count()) === 1);
await shot(p, "home-demos");

await browser.close();
finish();
