// Players panel: everyone's status, chips in and out, settling up with the banker, and copying the ledger.
import { URL, check, confirmCreate, finish, launch, shot } from "./lib.mjs";
const has = (p, sel, ms = 3000) => p.waitForSelector(sel, { timeout: ms }).then(() => true).catch(() => false);
const menu = async (p, item) => {
    await p.click('button[title="seat options"]');
    await p.click(`button:text-is("${item}")`);
};
// row is the text of name's line in the open players panel
const row = (p, name) =>
    p.locator('ul[aria-label="players"] li').filter({ has: p.locator(`span.font-semibold:text-is("${name}")`) }).innerText().catch(() => "");
const rowHas = async (p, name, text, ms = 3000) => {
    const end = Date.now() + ms;
    while (Date.now() < end) {
        if ((await row(p, name)).includes(text)) return true;
        await p.waitForTimeout(100);
    }
    return false;
};
const openPlayers = async (p) => {
    await p.click('button[aria-label^="chat"]');
    await p.click('button:text-is("players")');
};

const browser = await launch();
const alice = await (await browser.newContext({ permissions: ["clipboard-read", "clipboard-write"] })).newPage();
const bob = await (await browser.newContext()).newPage();
const dave = await (await browser.newContext()).newPage();
const carol = await (await browser.newContext({ viewport: { width: 375, height: 667 }, isMobile: true, hasTouch: true })).newPage();
const table = "players" + Date.now();
for (const [p, name, btn, seat] of [[alice, "alice", "create", 1], [bob, "bob", "join", 2], [dave, "dave", "join", 3]]) {
    await p.goto(URL);
    await p.fill("#username", name);
    await p.fill("#tablename", table);
    await p.click(`button:text-is("${btn}")`);
    if (btn === "create") await confirmCreate(p);
    await p.waitForSelector("text=Open");
    await p.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await p.locator("#buyIn ~ button").first().click();
    await p.waitForTimeout(300);
}
await carol.goto(`${URL}/?table=${table}`);
await carol.fill("#username", "carol");
await carol.press("#username", "Enter");
await carol.waitForSelector("text=watching");

await openPlayers(alice);
check("host is the banker", await rowHas(alice, "alice", "banker"));
check("seated players show seat and chips", await rowHas(alice, "bob", "playing · seat 2") && (await row(alice, "bob")).includes("stack 2000 · in 2000 · out 0"));
check("spectators are listed", await rowHas(alice, "carol", "watching"));
check("ledger starts balanced", await has(alice, "text=· balanced"));

// a rebuy adds to bought in
await menu(alice, "rebuy");
await alice.fill("#buyIn", "500");
await alice.press("#buyIn", "Enter");
check("rebuy counts as bought in", await rowHas(alice, "alice", "in 2500"));

// dave leaves the table and is listed as left with his cash-out
await dave.click('button:text-is("GoPoker")');
await dave.click('button:text-is("leave")');
check("a player who left is listed with their cash-out", await rowHas(alice, "dave", "left") && (await row(alice, "dave")).includes("out 2000"));
check("leaving even shows even", (await row(alice, "dave")).includes("even"));

// alice, first to act heads up, folds her small blind to bob
await alice.click('button:text-is("start")');
await alice.waitForSelector('button:text-is("fold")');
await alice.click('button:text-is("fold")');
check("winner requests from the banker", await rowHas(alice, "bob", "request 10 from alice"));
check("nets show signed", (await row(alice, "bob")).includes("+10") && (await row(alice, "alice")).includes("-10"));
check("still balanced after a hand", await has(alice, "text=· balanced"));

// bob's connection drops
await bob.close();
check("a dropped player shows as disconnected", await rowHas(alice, "bob", "disconnected", 5000));
await shot(alice, "players-panel");

await alice.click('button:text-is("copy ledger")');
const text = await alice.evaluate(() => navigator.clipboard.readText());
check("copied ledger names the banker", text.startsWith(`${table} ledger, in chips (banker: alice)`), text);
check("copied ledger has each line", /bob\s+\+10\s+request 10 from alice/.test(text) && /dave\s+0\s+even/.test(text), text);
check("copied ledger is balanced", text.endsWith("balanced"), text);

await openPlayers(carol);
check("spectators see the ledger", await rowHas(carol, "bob", "+10"));
check("panel fits a phone", !(await carol.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)));
await shot(carol, "players-panel-phone");
await browser.close();
finish();
