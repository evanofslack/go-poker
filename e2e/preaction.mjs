// Pre-action picks: what's offered, what a raise clears, and what plays on your turn.
import { check, finish, has, launch, quiet, seat, shot, until } from "./lib.mjs";

const picks = async (p) => (await p.locator('[aria-label="before your turn"] label').allInnerTexts()).map((t) => t.trim());
const pick = (p, label) => p.click(`label:text-is("${label}")`);
const picked = (p, label) => p.locator(`label:text-is("${label}") input`).isChecked();
const myTurn = async (p, label) => (await p.locator(`button:text-is("${label}"):enabled`).count()) === 1;

const browser = await launch();
const [alice, bob, carol] = await seat(browser, "pre" + Date.now(), ["alice", "bob", "carol"]);
await alice.click('button:text-is("start")');

// three handed, alice is on the button and acts first; bob is small blind, carol big blind
await alice.waitForSelector('button:text-is("call 20"):enabled');
check("small blind is offered fold / call 10", JSON.stringify(await picks(bob)) === '["fold","call 10"]', JSON.stringify(await picks(bob)));
check("big blind is offered check/fold / check", JSON.stringify(await picks(carol)) === '["check/fold","check"]', JSON.stringify(await picks(carol)));
check("waiting line stays with the picks", (await bob.locator("text=waiting for alice").count()) === 1);

await pick(bob, "call 10");
await pick(carol, "check");
check("a pick shows as checked", await picked(carol, "check"));
await pick(carol, "check/fold");
check("picking another unpicks the first", (await picked(carol, "check/fold")) && !(await picked(carol, "check")));
await pick(carol, "check/fold");
check("picking it again clears it", !(await picked(carol, "check/fold")));
await pick(carol, "check");
await shot(bob, "pre-1-picks");
await carol.setViewportSize({ width: 390, height: 844 });
await shot(carol, "pre-2-picks-phone");

await alice.click('button:text-is("raise")');
await alice.fill("#raise", "60");
await alice.press("#raise", "Enter");

check("a raise clears a call pick, and the turn waits for you", await has(bob, 'button:text-is("call 50"):enabled'));
check("a bet clears a check pick", (await has(carol, 'label:text-is("call 40")')) && !(await picked(carol, "call 40")));
check("it's offered fold / call 40", JSON.stringify(await picks(carol)) === '["fold","call 40"]', JSON.stringify(await picks(carol)));

await pick(carol, "call 40");
await bob.click('button:text-is("fold")');
check("a folded player sees just the waiting line", (await has(bob, "text=/^waiting for/")) && (await bob.locator("label").count()) === 0);

// carol's call plays, and after the flop carol acts first
check("a call pick plays on your turn", await has(carol, 'button:text-is("check"):enabled'));
await pick(alice, "check/fold");
await carol.click('button:text-is("check")');
// alice's pick checks for her, so the turn card comes and carol acts first again
check(
    "check/fold checks when it's free",
    await until(async () => (await picks(alice)).length === 2 && !(await picked(alice, "check/fold")) && (await myTurn(carol, "check")), 5000)
);

// after carol checks the turn, her pick is for the turn only
await carol.click('button:text-is("check")');
await carol.waitForSelector('label:text-is("check")');
await pick(carol, "check");
await alice.click('button:text-is("check")');
check("a pick doesn't carry to the next street", await until(() => myTurn(carol, "check"), 5000));
await quiet(1000);
check("so the turn waits for you", await myTurn(carol, "check"));

await pick(alice, "check/fold");
await carol.click('button:text-is("bet")');
await carol.press("#raise", "Enter");
check("check/fold folds facing a bet", await has(carol, "[data-won]"));

await browser.close();
finish();
