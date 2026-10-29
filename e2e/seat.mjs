// Seat menu: sit out / in, top up, sit out next hand, leave seat (asking first mid-hand).
import { btn, check, finish, fold, gone, has, launch, quiet, seat, seatMenu, shot } from "./lib.mjs";

const browser = await launch();
const [alice, bob] = await seat(browser, "seat" + Date.now(), ["alice", "bob"]);

check("only your own seat has a menu", (await alice.locator('button[title="seat options"]').count()) === 1);
await seatMenu(alice, "sit out");
check("sitting out shows to others", await has(bob, "text=sitting out"));
await alice.click('button[title="seat options"]');
await shot(alice, "seat-1-menu");
await alice.click(btn("sit in"));
check("sit in clears the label", await gone(bob, "text=sitting out"));

await alice.click('button[title="seat options"]');
check("with chips it's a top up, up to the max", await has(alice, `${btn("top up")}:has-text("up to 2,000")`));
await alice.click(btn("top up"));
check("top up form is capped at max minus stack", (await alice.inputValue("#buyIn")) === "2000", await alice.inputValue("#buyIn"));
await alice.click(btn("cancel"));
check("cancel goes back to the menu", await has(alice, btn("leave seat")));
await alice.click(btn("top up"));
await alice.fill("#buyIn", "500");
await alice.press("#buyIn", "Enter");
check("rebuy adds to the stack", await has(bob, "text=2500"));

await alice.click(btn("start"));
await bob.waitForSelector("text=/^(waiting for alice|fold)$/");
await alice.click('button[title="seat options"]');
check("mid-hand: top up waits, sit out next hand", (await alice.locator(btn("top up")).isDisabled()) && (await alice.locator(btn("sit out next hand")).count()) === 1);
await alice.click(btn("sit out next hand"));
await alice.click('button[title="seat options"]');
check("server confirms sit out next hand", await has(alice, btn("play next hand")));
check("leaving says it folds", await has(alice, `${btn("leave seat")}:has-text("folds this hand")`));
await alice.click(btn("leave seat"));
check("leaving mid-hand asks first", await has(alice, "[role=dialog] >> text=leave your seat?"));
await shot(alice, "seat-3-leave-confirm");
await alice.click(`[role=dialog] ${btn("cancel")}`);
check("cancel keeps the seat", await has(bob, 'p:text-is("alice")'));
await fold([alice, bob]);
check("sit-out applies when the hand ends", await has(bob, "text=sitting out", 4000));
await quiet(1000);
check("no next hand dealt while alice sits out", (await bob.locator(btn("fold")).count()) === 0);

await seatMenu(bob, "leave seat");
check("leaving removes bob for others", await gone(alice, 'p:text-is("bob")'));
check("bob sees open seats mid-game", await has(bob, "text=Open"));
await shot(alice, "seat-2-after-leave");

await browser.close();
finish();
