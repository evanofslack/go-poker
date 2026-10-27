// Stand-up game: the host starts a round, the winner shows to sit down, and the
// last one standing pays.
import { btn, check, finish, gone, has, launch, panel, seat, shot } from "./lib.mjs";

const browser = await launch();
const [alice, bob] = await seat(browser, "su" + Date.now(), ["alice", "bob"]);

// alice, hosting, starts a round for 2 big blinds from the host tab
await panel(alice, "host");
await shot(alice, "host-tab");
await alice.click('label:has(input[name="standUpPayout"]) >> text="2 bb"');
await alice.click(btn("start stand-up"));
check("the round waits for the next hand", await has(alice, "text=starts with the next hand"));
await alice.click('[aria-label="close table panel"]');
check("bob gets no host controls for it", (await bob.locator(btn("start stand-up")).count()) === 0);

await alice.click(btn("start"));
await alice.waitForSelector(btn("call 10"));
check("both start standing", (await bob.locator('text="standing"').count()) === 2);
await shot(bob, "standup-standing");

// alice folds, so bob wins without a call and has to show to sit
await alice.click(`${btn("fold")}:enabled`);
check("bob can show to sit down", await has(bob, btn("show to sit down")));
await bob.click(btn("show to sit down"));
check("alice pays as the last one standing", await has(alice, "text=alice pays the stand-up: 40 to 1 player"));
check("the round is over", await gone(bob, 'text="standing"'));

// the host can run it again from the footer, without stopping the game
check("alice can run it again", await has(alice, btn("stand-up again")));
check("only the host", (await bob.locator(btn("stand-up again")).count()) === 0);
await alice.click(btn("stand-up again"));
await panel(alice, "host");
check("a new round is set for the next hand", await has(alice, '[aria-label="stand-up round"] >> text=40 each'));
await shot(alice, "standup-host");

await browser.close();
finish();
