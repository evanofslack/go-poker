// Bomb pots and double board: the host calls a bomb pot, everyone posts and the
// hand starts on the flop; double board deals two boards and splits the pot.
import { btn, check, checkDown, finish, has, launch, panel, record, seat, shot } from "./lib.mjs";

const phone = { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 };
const browser = await launch();

// bomb pots of 5 bb, double board
const setup = async (p) => {
    await p.click('fieldset:has(input[name="bombPot"]) >> text="5 bb"');
    await p.click('label[for="bombDouble"]');
};
const [alice, bob, carol] = await seat(browser, "bomb" + Date.now(), ["alice", { name: "bob", context: phone }, "carol"], setup);
const rec = record(bob);
await bob.reload();

await panel(alice, "host");
await alice.click(btn("bomb pot next hand"));
check("the host sees it coming", await has(alice, '[aria-label="bomb pot"] >> text=next hand'));
await alice.click('[aria-label="close table panel"]');
check("everyone sees it coming", await has(bob, "[data-bomb] >> text=bomb pot next hand"));
check("only the host calls one", (await bob.locator(btn("bomb pot next hand")).count()) === 0);

await alice.click(btn("start"));
check("the flop is out", await has(bob, "[data-bomb] >> text=bomb pot · 100 each"));
const v = rec.views.at(-1);
check("it starts on the flop", v.street === "flop" && v.board.length === 3 && v.boards[1].length === 3, `${v.street} ${v.boards}`);
check("everyone put in 100", v.pots[0]?.amount === 300, JSON.stringify(v.pots));
check("two boards on the table", (await bob.locator("[data-run]").count()) === 1 && (await bob.locator("[data-run-caption]").first().innerText()).startsWith("board 1"));
check("a made hand for each board", (await bob.locator("[data-made-hand]").innerText()).includes("·"));
await shot(alice, "bomb-1-flop");
await shot(bob, "bomb-2-phone");

await checkDown([alice, bob, carol], async () => rec.hands.length === 1, 20000);
const h = rec.hands[0].record;
check("the record has the bomb and both boards", h.bomb === 100 && h.double && h.boards.length === 2, JSON.stringify(h.boards));
check("one pot a board", new Set(h.pots.map((p) => p.run ?? 0)).size === 2, JSON.stringify(h.pots));
await shot(alice, "bomb-3-showdown");
await shot(bob, "bomb-4-phone-showdown");

// the next hand is a normal one
check("then back to blinds", await has(bob, btn("fold"), 15000) && rec.views.at(-1).bomb === 0);
await panel(alice, "hands");
check("history says bomb, and board 2", await has(alice, "[data-history-run] >> text=board 2"));
await shot(alice, "bomb-5-history");

await browser.close();
finish();
