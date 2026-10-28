// Sounds, heard through the poker-sound events lib/sound.ts fires: the table's
// for players and watchers, your turn, the header's mute, a group switched off,
// chat from others, and a hidden tab keeping only your own.
import { btn, check, finish, fold, join, launch, open, panel, quick, quiet, seat, until } from "./lib.mjs";

// listen collects a page's sounds, now and after a reload, and lets the test
// hide its tab
async function listen(p) {
    const setup = () => {
        window.__sounds = [];
        window.addEventListener("poker-sound", (e) => window.__sounds.push(e.detail));
        Object.defineProperty(document, "hidden", { get: () => !!window.__hidden, configurable: true });
    };
    await p.addInitScript(setup);
    await p.evaluate(setup);
}
const heard = (p) => p.evaluate(() => window.__sounds.map((s) => s.group));
const hears = (p, group, ms) => until(async () => (await heard(p)).includes(group), ms);
const clear = (pages) => Promise.all(pages.map((p) => p.evaluate(() => (window.__sounds = []))));
const hide = (p, hidden) => p.evaluate((h) => (window.__hidden = h), hidden);

const browser = await launch();
const table = "sound" + Date.now();
const [alice, bob] = await seat(browser, table, ["alice", "bob"], quick);
const carol = await open(browser);
await join(carol, "carol", table);
const pages = [alice, bob];
for (const p of [alice, bob, carol]) await listen(p);

await alice.click(btn("start"));
check("a hand plays card and chip sounds", (await hears(alice, "cards")) && (await hears(alice, "chips")));
check("watchers hear the table", (await hears(carol, "cards")) && (await hears(carol, "chips")));
check("your turn plays the chime", await until(async () => (await heard(alice)).includes("turn") || (await heard(bob)).includes("turn")));
check("watchers have no turn", !(await heard(carol)).includes("turn"));

// muted, alice hears nothing while bob still does
await alice.click('button[aria-label="mute all sound"]');
await clear(pages);
await fold(pages);
check("unmuted players hear the fold", await hears(bob, "cards"));
await quiet(300);
check("mute plays nothing", (await heard(alice)).length === 0, (await heard(alice)).join(", "));
await alice.click('button[aria-label="mute all sound"]');

// chips off: the next hand deals with no chips for alice
await panel(alice, "options");
await alice.click('label:text-is("chips")');
await clear(pages);
check("the next hand still deals cards", await hears(alice, "cards", 6000));
await quiet(300);
check("a group switched off plays none of its sounds", !(await heard(alice)).includes("chips") && (await hears(bob, "chips")));
await alice.click('label:text-is("chips")');

// a hidden tab keeps your own sounds and drops the table's
await hide(alice, true);
await clear(pages);
await fold(pages);
check("a hidden tab still has the table for others", await hears(bob, "cards"));
await quiet(300);
check("a hidden tab drops the table's sounds", !(await heard(alice)).some((g) => g === "cards" || g === "chips"));
await hide(alice, false);

// chat plays for others' messages, not your own
await clear([alice, bob]);
await bob.click(btn("chat"));
await bob.fill("#input", "hi");
await bob.press("#input", "Enter");
check("chat plays for others' messages", await hears(alice, "chat"));
await quiet(300);
check("your own message plays nothing", !(await heard(bob)).includes("chat"));

// the test buttons play even with the group off
await alice.click('label:text-is("chat")');
await clear([alice]);
await alice.click('button[aria-label="play chat"]');
check("a test button plays its group", await hears(alice, "chat", 1000));

await browser.close();
finish();
