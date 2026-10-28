// The options tab: the viewer's own animations, mute and keyboard shortcuts,
// kept in this browser across reloads, and carried over from the old settings.
import { btn, check, finish, has, launch, open, join, panel, quiet, seat, shot } from "./lib.mjs";

const motion = (p) => p.locator("[data-motion]").getAttribute("data-motion").catch(() => null);
const pick = (p, name, value) => p.click(`fieldset:has(input[name="${name}"]) >> text="${value}"`);
const on = (p, id) => p.locator(`#${id}`).isChecked();

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

await alice.click('button[aria-label="mute all sound"]');
check("the header's mute shows in the tab", await on(alice, "muted"));
await alice.click('label:text-is("keyboard shortcuts")');
check("shortcuts off", !(await on(alice, "shortcuts")));

await alice.reload();
await alice.waitForSelector('p:text-is("alice")');
await quiet(700);
await panel(alice, "options");
check("animations stay on after a reload", (await motion(alice)) === "on");
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

await browser.close();
finish();
