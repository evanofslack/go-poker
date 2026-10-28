// Players panel: everyone's status, chips in and out, settling up with the banker, and copying the ledger.
import { URL, btn, check, finish, has, launch, open, panel, playerRow, seat, seatMenu, shot, until } from "./lib.mjs";

const rowHas = (p, name, text, ms) => until(async () => (await playerRow(p, name)).includes(text), ms);

const browser = await launch();
const table = "players" + Date.now();
const clipboard = { permissions: ["clipboard-read", "clipboard-write"] };
const [alice, bob, dave] = await seat(browser, table, [{ name: "alice", context: clipboard }, "bob", "dave"]);
const carol = await open(browser, { viewport: { width: 375, height: 667 }, isMobile: true, hasTouch: true });
await carol.goto(`${URL}/?table=${table}`);
await carol.fill("#username", "carol");
await carol.press("#username", "Enter");
await carol.waitForSelector("text=Open");

await panel(alice, "players");
check("host is the banker", await rowHas(alice, "alice", "banker"));
check("seated players show seat and chips", (await rowHas(alice, "bob", "playing · seat 2")) && (await playerRow(alice, "bob")).includes("stack 2000 · in 2000 · out 0"));
check("spectators are listed", await rowHas(alice, "carol", "watching"));
check("ledger starts balanced", await has(alice, "text=· balanced"));

// a rebuy adds to bought in
await seatMenu(alice, "rebuy");
await alice.fill("#buyIn", "500");
await alice.press("#buyIn", "Enter");
check("rebuy counts as bought in", await rowHas(alice, "alice", "in 2500"));

// dave leaves the table and is listed as left with his cash-out
await dave.click(btn("GoPoker"));
await dave.click(btn("leave"));
check("a player who left is listed with their cash-out", (await rowHas(alice, "dave", "left")) && (await playerRow(alice, "dave")).includes("out 2000"));
check("leaving even shows even", (await playerRow(alice, "dave")).includes("even"));

// alice, first to act heads up, folds her small blind to bob
await alice.click(btn("start"));
await alice.click(`${btn("fold")}:enabled`);
check("winner requests from the banker", await rowHas(alice, "bob", "request 10 from alice"));
check("nets show signed", (await playerRow(alice, "bob")).includes("+10") && (await playerRow(alice, "alice")).includes("-10"));
check("still balanced after a hand", await has(alice, "text=· balanced"));

// bob's connection drops
await bob.close();
check("a dropped player shows as disconnected", await rowHas(alice, "bob", "disconnected", 5000));
await shot(alice, "players-panel");

await alice.click(btn("copy ledger"));
const text = await alice.evaluate(() => navigator.clipboard.readText());
check("copied ledger names the banker", text.startsWith(`${table} ledger, in chips (banker: alice)`), text);
check("copied ledger has each line", /bob\s+\+10\s+request 10 from alice/.test(text) && /dave\s+0\s+even/.test(text), text);
check("copied ledger is balanced", text.endsWith("balanced"), text);

await panel(carol, "players");
check("spectators see the ledger", await rowHas(carol, "bob", "+10"));
check("panel fits a phone", !(await carol.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)));
await shot(carol, "players-panel-phone");
await browser.close();
finish();
