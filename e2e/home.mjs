// The home page on desktop and phone, dark and light: the join form fills the first screen and
// the features section waits below it.
import { URL, check, finish, launch, open, shot } from "./lib.mjs";

const views = {
    desktop: { viewport: { width: 1280, height: 720 } },
    phone: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

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
                features: rect('[aria-label="features"]').top,
                height: window.innerHeight,
                wide: document.documentElement.scrollWidth > window.innerWidth,
            };
        });
        const name = `${kind}, ${colorScheme}`;
        check(`${name}: form on the first screen`, box.form <= box.height, `bottom ${box.form}`);
        check(`${name}: features below the fold`, box.features >= box.height, `top ${box.features}`);
        check(`${name}: no sideways scroll`, !box.wide);
        check(`${name}: six features`, (await p.locator('[aria-label="features"] li').count()) === 6);
        await shot(p, `home-${kind}-${colorScheme}`);
        await p.locator('[aria-label="features"]').scrollIntoViewIfNeeded();
        await shot(p, `home-features-${kind}-${colorScheme}`);
        await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
        await shot(p, `home-end-${kind}-${colorScheme}`);
        await p.context().close();
    }
}
await browser.close();
finish();
