// The options tab: the viewer's own animations, looks, mute and keyboard shortcuts,
// kept in this browser across reloads, and carried over from the old settings.
import { btn, check, finish, has, launch, open, join, panel, quiet, seat, shot } from "./lib.mjs";

const motion = (p) => p.locator("[data-motion]").getAttribute("data-motion").catch(() => null);
const pick = (p, name, value) => p.click(`fieldset:has(input[name="${name}"]) >> text="${value}"`);
const on = (p, id) => p.locator(`#${id}`).isChecked();
const look = (p, name) => p.evaluate((n) => document.documentElement.dataset[n], name);
const color = (p, v) => p.evaluate((n) => getComputedStyle(document.documentElement).getPropertyValue(n).trim(), v);
const rail = (p) => p.locator(".felt").evaluate((e) => getComputedStyle(e).borderTopColor);

const browser = await launch();
const table = "opt" + Date.now();
const [alice] = await seat(browser, table, ["alice", "bob"]);

await quiet(700);
check("animations follow the system by default", (await motion(alice)) === "system");
await panel(alice, "options");
check("the tab has display, sound and keys", (await has(alice, 'section[aria-label="display"]')) && (await has(alice, 'section[aria-label="sound"]')) && (await has(alice, 'section[aria-label="keys"]')));
check("the settings tab no longer has the keys", await (async () => {
    await panel(alice, "settings");
    const gone = (await alice.locator('[aria-label="keyboard shortcuts"]').count()) === 0;
    await panel(alice, "options");
    return gone;
})());
await shot(alice, "options-1");

await pick(alice, "motion", "off");
check("animations off", (await motion(alice)) === "off");
await pick(alice, "motion", "on");
check("animations on", (await motion(alice)) === "on");

check("four colors, green felt and red backs by default", (await look(alice, "deck")) === "four" && (await look(alice, "felt")) === "green" && (await look(alice, "back")) === "red");
await pick(alice, "deck", "two color");
check("two colors: diamonds take the heart color", (await color(alice, "--color-diamond")) === (await color(alice, "--color-heart")));
check("two colors: clubs take the spade color", (await color(alice, "--color-club")) === (await color(alice, "--color-spade")));
const green = await rail(alice);
await pick(alice, "felt", "blue");
check("a blue table changes the rail", (await look(alice, "felt")) === "blue" && (await rail(alice)) !== green);
const red = await color(alice, "--color-card-back");
await pick(alice, "back", "black");
check("black backs change the back color", (await look(alice, "back")) === "black" && (await color(alice, "--color-card-back")) !== red);

await alice.click('button[aria-label="mute all sound"]');
check("the header's mute shows in the tab", await on(alice, "muted"));
await alice.click('label:text-is("keyboard shortcuts")');
check("shortcuts off", !(await on(alice, "shortcuts")));

await alice.reload();
await alice.waitForSelector('p:text-is("alice")');
await quiet(700);
await panel(alice, "options");
check("animations stay on after a reload", (await motion(alice)) === "on");
check("looks stay after a reload", (await look(alice, "deck")) === "two" && (await look(alice, "felt")) === "blue" && (await look(alice, "back")) === "black");
check("mute stays on after a reload", await on(alice, "muted"));
check("shortcuts stay off after a reload", !(await on(alice, "shortcuts")));
check("the header's button says unmute", (await alice.locator('button[title="unmute"]').count()) === 1);

// someone who muted and turned shortcuts off before options existed keeps both
const carol = await open(browser);
await carol.addInitScript(() => {
    if (!localStorage.getItem("prefs")) {
        localStorage.setItem("muted", "1");
        localStorage.setItem("shortcuts", "0");
    }
});
await join(carol, "carol", table);
await panel(carol, "options");
check("an old mute carries over", await on(carol, "muted"));
check("old shortcuts off carry over", !(await on(carol, "shortcuts")));

// each table and back, dark and light, with cards dealt
if (process.env.SHOTS) {
    await alice.click(btn("start"));
    await alice.waitForTimeout(800);
    for (const scheme of ["dark", "light"]) {
        await alice.emulateMedia({ colorScheme: scheme });
        await alice.waitForTimeout(300);
        await shot(alice, `options-${scheme}-swatches`);
        for (const [felt, back] of [["green", "red"], ["yellow", "violet"], ["white", "black"], ["indigo", "white"]]) {
            await pick(alice, "felt", felt);
            await pick(alice, "back", back);
            await alice.click('[aria-label="table"] button[aria-label="close table panel"]');
            await alice.waitForTimeout(300);
            await shot(alice, `options-${scheme}-${felt}-${back}`);
            await panel(alice, "options");
        }
    }
}

await browser.close();
finish();
