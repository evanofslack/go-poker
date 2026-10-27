// Seat menu: sit out / in, rebuy, sit out next hand, leave seat.
import { check, finish, fold, gone, has, launch, quiet, seat, seatMenu, shot } from "./lib.mjs";

const browser = await launch();
const [alice, bob] = await seat(browser, "seat" + Date.now(), ["alice", "bob"]);

check("only your own seat has a menu", (await alice.locator('button[title="seat options"]').count()) === 1);
await seatMenu(alice, "sit out");
check("sitting out shows to others", await has(bob, "text=sitting out"));
await alice.click('button[title="seat options"]');
await shot(alice, "seat-1-menu");
await alice.click('button:text-is("sit in")');
check("sit in clears the label", await gone(bob, "text=sitting out"));

await seatMenu(alice, "rebuy");
check("rebuy form is capped at max minus stack", (await alice.inputValue("#buyIn")) === "2000", await alice.inputValue("#buyIn"));
await alice.fill("#buyIn", "500");
await alice.press("#buyIn", "Enter");
check("rebuy adds to the stack", await has(bob, "text=2500"));

await alice.click('button:text-is("start")');
await bob.waitForSelector("text=/^(waiting for alice|fold)$/");
await alice.click('button[title="seat options"]');
check("mid-hand: no rebuy, sit out next hand", (await alice.locator('button:text-is("rebuy")').count()) === 0 && (await alice.locator('button:text-is("sit out next hand")').count()) === 1);
await alice.click('button:text-is("sit out next hand")');
await alice.click('button[title="seat options"]');
check("server confirms sit out next hand", await has(alice, 'button:text-is("play next hand")'));
await alice.click('button[title="seat options"]');
await fold([alice, bob]);
check("sit-out applies when the hand ends", await has(bob, "text=sitting out", 4000));
await quiet(1000);
check("no next hand dealt while alice sits out", (await bob.locator('button:text-is("fold")').count()) === 0);

await seatMenu(bob, "leave seat");
check("leaving removes bob for others", await gone(alice, 'p:text-is("bob")'));
check("bob sees open seats mid-game", await has(bob, "text=Open"));
await shot(alice, "seat-2-after-leave");

await browser.close();
finish();
