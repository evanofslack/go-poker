// Keyboard shortcuts for the betting controls.
import { check, finish, has, launch, panel, quiet, seat, shot } from "./lib.mjs";

const picked = (p, label) => p.locator(`label:text-is("${label}") input`).isChecked();
const myTurn = async (p, label) => (await p.locator(`button:text-is("${label}"):enabled`).count()) === 1;
// the buttons ignore input for a moment as they appear
const settle = () => quiet(400);

const browser = await launch();
const [alice, bob] = await seat(browser, "keys" + Date.now(), ["alice", "bob"]);
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("call 10"):enabled');
await settle();

// heads up, alice is the small blind and acts first
await bob.keyboard.press("c");
check("c picks check before your turn", await picked(bob, "check"));
await bob.keyboard.press("c");
check("c again clears it", !(await picked(bob, "check")));
await bob.keyboard.press("f");
check("f picks check/fold", await picked(bob, "check/fold"));
await bob.keyboard.press("f");
await shot(alice, "keys-1-hints");

await alice.keyboard.press("c");
check("c calls on your turn", await has(bob, 'button:text-is("check"):enabled'));
await settle();
await bob.keyboard.press("f");
await quiet(500);
check("f doesn't fold when checking is free", await myTurn(bob, "check"));

await bob.keyboard.press("r");
check("r opens the raise panel", await has(bob, "#raise"));
await bob.keyboard.press("ArrowUp");
check("up adds a big blind", (await bob.inputValue("#raise")) === "60", await bob.inputValue("#raise"));
await bob.keyboard.press("ArrowDown");
await bob.keyboard.press("ArrowDown");
check("down can't go below the min raise", (await bob.inputValue("#raise")) === "40", await bob.inputValue("#raise"));
await bob.keyboard.press("ArrowUp");
await bob.keyboard.press("Enter");
check("enter raises", await has(alice, 'button:text-is("call 40"):enabled'));
await settle();

await alice.click('button:text-is("chat")');
await alice.click("#input");
await alice.keyboard.type("fcr");
await quiet(500);
check("typing in chat isn't a shortcut", (await myTurn(alice, "call 40")) && (await alice.inputValue("#input")) === "fcr");
await alice.click('button:text-is("chat")');

await panel(alice, "settings");
check("the settings tab lists the keys", await has(alice, '[aria-label="keyboard shortcuts"]'));
await shot(alice, "keys-2-settings");
await alice.click('label:text-is("keyboard shortcuts")');
await alice.keyboard.press("c");
await quiet(500);
check("turned off, keys do nothing", await myTurn(alice, "call 40"));
await alice.click('label:text-is("keyboard shortcuts")');
await alice.keyboard.press("c");
check("turned back on, c calls", await has(bob, 'button:text-is("check"):enabled'));

await browser.close();
finish();
