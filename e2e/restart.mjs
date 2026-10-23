// Tables survive a server restart: a graceful one mid-hand carries the hand on,
// and a crash goes back to the last hand's end. Runs its own server on :8097
// with a SQLite file; BIN names the server binary (just e2e sets it).
import { spawn } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { check, confirmCreate, finish, launch } from "./lib.mjs";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

if (!process.env.BIN) {
    console.log("BIN must name a server binary; just e2e sets it");
    process.exit(1);
}
const PORT = 8097;
const URL = `http://localhost:${PORT}`;
const db = join(mkdtempSync(join(tmpdir(), "poker-")), "poker.db");

let server;
async function start() {
    server = spawn(process.env.BIN, [], {
        env: { ...process.env, PORT: String(PORT), STATIC_DIR: "../web/out", DATABASE_URL: "sqlite:" + db, METRICS_ADDR: "" },
        stdio: "ignore",
    });
    for (let i = 0; i < 50; i++) {
        if (await fetch(URL).then((r) => r.ok, () => false)) return;
        await sleep(200);
    }
    throw new Error("server didn't start");
}

function stop(signal) {
    return new Promise((resolve) => {
        server.once("exit", resolve);
        server.kill(signal);
    });
}

// record keeps the views, welcomes and finished hands a page receives
function record(page) {
    const rec = { views: [], welcomes: 0, done: [] };
    page.on("websocket", (ws) => {
        ws.on("framereceived", ({ payload }) => {
            try {
                const m = JSON.parse(payload);
                if (m.action === "welcome") rec.welcomes++;
                if (m.action === "welcome" || m.action === "table-update") rec.views.push(m.view);
                for (const e of m.events ?? []) if (e.type === "hand-done") rec.done.push(e.data);
            } catch {}
        });
    });
    return rec;
}

async function waitFor(cond, ms, what) {
    const end = Date.now() + ms;
    while (Date.now() < end) {
        if (cond()) return true;
        await sleep(100);
    }
    console.log(`  timed out waiting for ${what}`);
    return false;
}

const last = (rec) => rec.views.at(-1);
const seats = (v) => v?.players.map((p) => `${p.name}@${p.seat}`).join(",");

async function enter(page, name, table, button) {
    await page.goto(URL);
    await page.fill("#username", name);
    await page.fill("#tablename", table);
    await page.click(`button:text-is("${button}")`);
    if (button === "create") await confirmCreate(page);
}

async function sit(page, seat) {
    await page.locator("button", { hasText: "Open" }).filter({ hasText: String(seat) }).click();
    await page.locator("#buyIn ~ button").first().click();
}

// restart stops the server with signal and starts it again, waiting for both
// pages to rejoin
async function restart(signal, recs) {
    const before = recs.map((r) => r.welcomes);
    await stop(signal);
    await start();
    return waitFor(() => recs.every((r, i) => r.welcomes > before[i]), 20000, "both players to rejoin");
}

await start();
const browser = await launch();
const alice = await (await browser.newContext()).newPage();
const bob = await (await browser.newContext()).newPage();
const aRec = record(alice);
const bRec = record(bob);
const TABLE = "restart" + Date.now();

await enter(alice, "alice", TABLE, "create");
await alice.waitForSelector("text=Open");
await sit(alice, 1);
await waitFor(() => last(aRec)?.players.length === 1, 3000, "alice seated");
await enter(bob, "bob", TABLE, "join");
await bob.waitForSelector("text=Open");
await sit(bob, 2);
await waitFor(() => last(aRec)?.players.length === 2, 3000, "bob seated");
await alice.click('button:text-is("start")');
await waitFor(() => last(aRec)?.betting, 3000, "the first hand");

// a graceful restart mid-hand carries the hand on
const mid = last(aRec);
const aliceCards = (v) => JSON.stringify(v?.players.find((p) => p.name === "alice")?.cards);
check("rejoined after a graceful restart", await restart("SIGTERM", [aRec, bRec]));
const back = last(aRec);
check("seats kept", seats(back) === seats(mid), `${seats(mid)} then ${seats(back)}`);
check("the hand carries on", back.hand === mid.hand && back.street === mid.street && aliceCards(back) === aliceCards(mid), `hand ${mid.hand} then ${back.hand}`);
check("still at the table", (await alice.locator("#username").count()) === 0 && (await bob.locator("#username").count()) === 0);

// finish the hand, and crash once the next one is dealt
for (let i = 0; i < 20 && aRec.done.length === 0; i++) {
    for (const page of [alice, bob]) {
        await page.click('button:text-is("fold"):enabled', { timeout: 300 }).catch(() => {});
    }
}
await waitFor(() => aRec.done.length > 0, 5000, "the hand to end");
const ended = aRec.done.at(-1);
await waitFor(() => last(aRec)?.hand === ended.record.hand + 1 && last(aRec)?.betting, 10000, "the next hand");
check("rejoined after a crash", await restart("SIGKILL", [aRec, bRec]));
await waitFor(() => last(aRec)?.betting, 5000, "the hand to be dealt again");
const after = last(aRec);
const total = (v) => Object.fromEntries(v.players.map((p) => [p.name, p.stack + p.committed]));
check("seats kept after a crash", seats(after) === seats(mid), seats(after));
check("stacks back to the last hand's end", JSON.stringify(total(after)) === JSON.stringify(ended.stats.stacks), `${JSON.stringify(ended.stats.stacks)} then ${JSON.stringify(total(after))}`);
check("the lost hand is dealt again", after.hand === ended.record.hand + 1, `hand ${after.hand}`);

await browser.close();
await stop("SIGTERM");
finish();
