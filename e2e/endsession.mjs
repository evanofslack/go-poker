// Host session controls: pause and resume, ending the session after the hand, and the final settle-up.
import { URL, btn, check, finish, gone, has, launch, open, panel, seat, shot } from "./lib.mjs";

const hostTab = `[aria-label="table"] ${btn("host")}`;
const hostMenu = async (p, item) => {
    await panel(p, "host");
    await p.click(`[aria-label="table"] ${btn(item)}`);
};

const browser = await launch();
const table = "end" + Date.now();
const [alice, bob] = await seat(browser, table, ["alice", "bob"]);
await panel(alice, "players");
await panel(bob, "players");
check("host sees the host tab", (await alice.locator(hostTab).count()) === 1);
check("non-host doesn't see it", (await bob.locator(hostTab).count()) === 0);
await bob.keyboard.press("Escape");

// pausing before the start blocks it
await hostMenu(alice, "pause");
check("everyone sees the pause", await has(bob, "text=paused by the host"));
check("start is disabled while paused", await has(alice, `${btn("start")}:disabled`));
await hostMenu(alice, "resume");
check("resuming clears the hint", await gone(bob, "text=paused by the host"));
await alice.click(btn("start"));
await alice.waitForSelector(btn("fold"));

// ending mid-hand waits for the hand
await hostMenu(alice, "end session");
check("ending asks first", await has(alice, "[role=dialog] >> text=after this hand"));
await alice.click(`[role=dialog] ${btn("end session")}`);
check("everyone is told it's the last hand", await has(bob, "text=last hand"));
check("the host tab drops end session once ending", await gone(alice, `[aria-label="table"] ${btn("end session")}`));
await alice.keyboard.press("Escape");
await alice.click(btn("fold"));

check("everyone gets the settle-up", await has(bob, "[role=dialog] >> text=the session is over"));
const final = await bob.locator('ul[aria-label="final ledger"]').innerText().catch(() => "");
check("it shows who sends and gets what", final.includes("alice") && final.includes("-10") && final.includes("+10"), final);
await shot(bob, "session-ended");
await bob.click(`[role=dialog] ${btn("close")}`);
check("the table says when it closes", await has(bob, "text=table closes in 60m"));
check("open seats are gone", (await bob.locator("button", { hasText: "Open" }).count()) === 0);
check("no more start", (await alice.locator(btn("start")).count()) === 0);
await alice.click(`[role=dialog] ${btn("close")}`);
await panel(alice, "players");
check("the host tab is gone", (await alice.locator(hostTab).count()) === 0);

// a late spectator can still read it
const carol = await open(browser);
await carol.goto(URL);
await carol.fill("#username", "carol");
await carol.fill("#tablename", table);
await carol.click(btn("join"));
check("a late joiner sees the settle-up", await has(carol, "[role=dialog] >> text=the session is over"));
await browser.close();
finish();
