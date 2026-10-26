// Rebuy approval: rebuys and sitting back down wait for the host to approve or deny.
import { check, finish, gone, has, launch, seat, seatMenu, shot, sit } from "./lib.mjs";

const browser = await launch();
const [alice, bob] = await seat(browser, "approve" + Date.now(), ["alice", "bob"], (p) => p.check("#approveRebuys"));
check("a first buy-in needs no approval", await has(alice, 'p:text-is("bob")'));

const requests = 'ul[aria-label="buy-in requests"]';
await seatMenu(bob, "rebuy");
await bob.fill("#buyIn", "500");
await bob.press("#buyIn", "Enter");
check("the host sees the request", await has(alice, `${requests} >> text=wants to rebuy for 500`));
check("the player sees it's waiting", await has(bob, "text=waiting for the host to ok your buy-in of 500"));
check("others can't approve", (await bob.locator(requests).count()) === 0);
await shot(alice, "approval-request");
await alice.click(`${requests} button:text-is("deny")`);
check("deny clears the request", await gone(bob, "text=waiting for the host"));
check("a denied rebuy adds nothing", !(await has(alice, "text=2500", 500)));

await seatMenu(bob, "rebuy");
await bob.fill("#buyIn", "500");
await bob.press("#buyIn", "Enter");
await alice.click(`${requests} button:text-is("approve")`);
check("an approved rebuy lands", await has(bob, "text=2500"));

// leaving and sitting back down counts as a rebuy
await seatMenu(bob, "leave seat");
await bob.waitForSelector("text=Open");
await sit(bob, 3);
check("sitting back down asks the host", await has(alice, `${requests} >> text=wants to sit back down`));
check("and doesn't seat them yet", !(await has(alice, 'p:text-is("bob")', 500)));
await alice.click(`${requests} button:text-is("approve")`);
check("approving seats them", await has(alice, 'p:text-is("bob")'));
await browser.close();
finish();
