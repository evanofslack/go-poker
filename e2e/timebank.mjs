// Time bank: the add-time button, cards running out, and a card spent when the turn timer runs out.
import { btn, check, finish, has, gone, join, last, launch, open, quick, record, seat, shot, sit, until } from "./lib.mjs";

const browser = await launch();
const table = "timebank" + Date.now();
// a 15s turn, one card of 10s each
const setup = async (p) => {
    await p.click('fieldset:has(input[name="turnTime"]) >> text="15s"');
    await p.click('fieldset:has(input[name="timeBank"]) >> text="1"');
    await p.click('fieldset:has(input[name="timeCard"]) >> text="10s"');
    await quick(p);
};
const [alice] = await seat(browser, table, ["alice"], setup);
const bob = await open(browser);
const bRec = record(bob);
await join(bob, "bob", table);
await sit(bob, 2);
await alice.click(btn("start"));
await alice.waitForSelector(btn("call 10"));

const add = btn("+10s");
check("your turn offers the time bank", await has(alice, add));
check("it isn't offered off your turn", (await bob.locator(add).count()) === 0);
await shot(alice, "timebank-1-button");
const ends = () => last(bRec)?.turnEndsIn ?? 0;
const before = ends();
await alice.click(add);
check("adding time moves the turn timer", await until(() => ends() > before + 5000), `${before} -> ${ends()}`);
check("the last card is gone with it", await gone(alice, add));
const cards = (name) => last(bRec)?.players.find((p) => p.name === name)?.timeCards;
check("everyone sees the cards left", await until(() => cards("alice") === 0), String(cards("alice")));

// bob lets his turn run out: his card goes first, then the timer acts
await alice.click(btn("call 10"));
await bob.waitForSelector(btn("+10s"));
check("a card is spent when time runs out", await until(() => cards("bob") === 0, 17000), String(cards("bob")));
check("bob still has the turn", !!last(bRec)?.legal.turn);
await shot(bob, "timebank-2-spent");

await browser.close();
finish();
