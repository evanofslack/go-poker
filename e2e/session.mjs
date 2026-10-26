// Two-player session: hidden cards, next hand, reload, turn timer and grace period.
import { URL, check, create, finish, gone, has, join, last, launch, open, quiet, record, shot, sit, until } from "./lib.mjs";

// dealt counts the hands dealt, from updates where betting starts
const dealt = (rec) => rec.views.filter((g, i) => g.betting && !rec.views[i - 1]?.betting).length;
const cardsShown = (g, name) => g.players.find((p) => p.name === name)?.cards[0] !== 0;

const table = "e2e" + Date.now();
const browser = await launch();
const alice = await open(browser);
const bob = await open(browser);
const aRec = record(alice);
const bRec = record(bob);

// create, join, sit, start
await create(alice, "alice", table);
await sit(alice, 1);
await until(() => last(aRec)?.players.length === 1);
await join(bob, "bob", table);
await sit(bob, 2);
await until(() => last(aRec)?.players.length === 2);

// a third tab can't take a name that's in use
const eve = await open(browser);
await eve.goto(URL);
await eve.fill("#username", "ALICE");
await eve.fill("#tablename", table);
await eve.click('button:text-is("join")');
check("duplicate name rejected on join form", await has(eve, "text=name already taken"));
await shot(eve, "1-name-taken");
await eve.context().close();

await alice.click('button:text-is("start")');
await until(() => last(aRec)?.betting && last(bRec)?.betting);
await shot(alice, "2-alice-hand1");
await shot(bob, "3-bob-hand1");

// hidden cards: every frame before the first showdown
const preShowdown = (rec) => rec.views.filter((g) => g.running && g.street !== "");
check("alice never receives bob's cards mid-hand", preShowdown(aRec).every((g) => !cardsShown(g, "bob")) && cardsShown(last(aRec), "alice"));
check("bob never receives alice's cards mid-hand", preShowdown(bRec).every((g) => !cardsShown(g, "alice")) && cardsShown(last(bRec), "bob"));

// alice (dealer, small blind) folds; the server records it and deals the next hand once
await alice.click('button:text-is("fold")');
await until(() => dealt(aRec) >= 2, 8000);
await quiet(1000);
check("server deals the next hand exactly once", dealt(aRec) === 2, `dealt ${dealt(aRec)}`);
const first = aRec.hands[0]?.record;
check(
    "fold and win recorded by the server",
    first?.actions.at(-1)?.name === "alice" && first?.actions.at(-1)?.kind === "fold" && first?.pots[0]?.winners[0] === "bob",
    JSON.stringify(first),
);
check("the log leaves out poker actions", !aRec.logs.some((l) => /folds|wins|starting new hand|blind/.test(l)), JSON.stringify(aRec.logs));

// reload bob's tab mid-hand: it rejoins and keeps the seat
const bobStack = last(aRec).players.find((p) => p.name === "bob").stack;
const welcomes = bRec.welcomes;
await bob.reload();
const rejoined = await until(() => bRec.welcomes > welcomes && last(bRec).players.some((p) => p.name === "bob") && cardsShown(last(bRec), "bob"), 5000);
check("reload reclaims the seat with its cards", rejoined && (await gone(bob, "#username")));
check("stack kept after reload", last(aRec).players.find((p) => p.name === "bob").stack === bobStack);
await shot(bob, "4-bob-after-reload");

// close bob's tab; bob acts first in hand 2 (dealer), so the turn timer acts for him
await bob.context().close();
console.log("  waiting for the 30s turn timer...");
const timedOut = await until(() => aRec.hands.some((h) => h.record.actions.some((a) => a.name === "bob" && a.timeout)), 40000);
check("turn timer acts for a closed tab", timedOut, JSON.stringify(aRec.hands.at(-1)?.record.actions));
console.log("  waiting for the 60s grace period...");
check("closed tab is removed after the grace period", await until(() => aRec.logs.includes("bob left the table"), 70000));
await shot(alice, "5-alice-after-bob-left");

await browser.close();
finish();
