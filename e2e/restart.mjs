// Tables survive a server restart: a graceful one mid-hand carries the hand on,
// and a crash goes back to the last hand's end. Runs its own server on :8097
// with a SQLite file; BIN names the server binary (just e2e sets it).
import { spawn } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { check, create, finish, fold, join, last, launch, open, record, sit, until } from "./lib.mjs";

if (!process.env.BIN) {
    console.log("BIN must name a server binary; just e2e sets it");
    process.exit(1);
}
const PORT = 8097;
const URL = `http://localhost:${PORT}`;
const db = path.join(mkdtempSync(path.join(tmpdir(), "poker-")), "poker.db");

let server;
async function start() {
    server = spawn(process.env.BIN, [], {
        env: { ...process.env, PORT: String(PORT), STATIC_DIR: "../web/out", DATABASE_URL: "sqlite:" + db, METRICS_ADDR: "" },
        stdio: "ignore",
    });
    for (let i = 0; i < 50; i++) {
        if (await fetch(URL).then((r) => r.ok, () => false)) return;
        await new Promise((r) => setTimeout(r, 200));
    }
    throw new Error("server didn't start");
}

function stop(signal) {
    return new Promise((resolve) => {
        server.once("exit", resolve);
        server.kill(signal);
    });
}

const seats = (v) => v?.players.map((p) => `${p.name}@${p.seat}`).join(",");

// restart stops the server with signal and starts it again, waiting for both
// pages to rejoin
async function restart(signal, recs) {
    const before = recs.map((r) => r.welcomes);
    await stop(signal);
    await start();
    return until(() => recs.every((r, i) => r.welcomes > before[i]), 20000);
}

await start();
const browser = await launch();
const alice = await open(browser);
const bob = await open(browser);
const aRec = record(alice);
const bRec = record(bob);
const table = "restart" + Date.now();

for (const p of [alice, bob]) await p.goto(URL);
await create(alice, "alice", table);
await sit(alice, 1);
await join(bob, "bob", table);
await sit(bob, 2);
await until(() => last(aRec)?.players.length === 2);
await alice.click('button:text-is("start")');
await until(() => last(aRec)?.betting);

// a graceful restart mid-hand carries the hand on
const mid = last(aRec);
const aliceCards = (v) => JSON.stringify(v?.players.find((p) => p.name === "alice")?.cards);
check("rejoined after a graceful restart", await restart("SIGTERM", [aRec, bRec]));
const back = last(aRec);
check("seats kept", seats(back) === seats(mid), `${seats(mid)} then ${seats(back)}`);
check("the hand carries on", back.hand === mid.hand && back.street === mid.street && aliceCards(back) === aliceCards(mid), `hand ${mid.hand} then ${back.hand}`);
check("still at the table", (await alice.locator("#username").count()) === 0 && (await bob.locator("#username").count()) === 0);

// finish the hand, and crash once the next one is dealt
await fold([alice, bob]);
await until(() => aRec.hands.length > 0, 5000);
const ended = aRec.hands.at(-1);
await until(() => last(aRec)?.hand === ended.record.hand + 1 && last(aRec)?.betting, 10000);
check("rejoined after a crash", await restart("SIGKILL", [aRec, bRec]));
await until(() => last(aRec)?.betting, 5000);
const after = last(aRec);
const total = (v) => Object.fromEntries(v.players.map((p) => [p.name, p.stack + p.committed]));
check("seats kept after a crash", seats(after) === seats(mid), seats(after));
check("stacks back to the last hand's end", JSON.stringify(total(after)) === JSON.stringify(ended.stats.stacks), `${JSON.stringify(ended.stats.stacks)} then ${JSON.stringify(total(after))}`);
check("the lost hand is dealt again", after.hand === ended.record.hand + 1, `hand ${after.hand}`);

await browser.close();
await stop("SIGTERM");
finish();
