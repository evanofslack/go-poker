// Runs the e2e scripts against a server already up, a few at a time, printing
// each script's output in one piece as it finishes: node run.mjs [script...]
import { spawn } from "node:child_process";
import { readdirSync } from "node:fs";

const jobs = Number(process.env.JOBS) || 4;
// the slowest go first, so they don't hold up the end
const slow = ["session", "bots", "layout", "history", "stats"];
// lobby checks the public table list, so nothing else runs alongside it
const alone = ["lobby"];

const all = readdirSync(".")
    .filter((f) => f.endsWith(".mjs") && !["lib.mjs", "run.mjs"].includes(f))
    .map((f) => f.slice(0, -4));
const named = process.argv.slice(2);
for (const s of named) {
    if (!all.includes(s)) {
        console.log(`no script ${s}.mjs`);
        process.exit(1);
    }
}
const scripts = (named.length ? named : all).sort((a, b) => rank(a) - rank(b));
function rank(s) {
    const i = slow.indexOf(s);
    return i === -1 ? slow.length : i;
}

// run runs one script, resolving to its name, time and whether it passed
function run(name) {
    const start = Date.now();
    return new Promise((resolve) => {
        const child = spawn(process.execPath, [`${name}.mjs`], { stdio: ["ignore", "pipe", "pipe"] });
        let out = "";
        child.stdout.on("data", (d) => (out += d));
        child.stderr.on("data", (d) => (out += d));
        child.on("exit", (code) => {
            const secs = Math.round((Date.now() - start) / 1000);
            console.log(`== ${name} (${secs}s)\n${out.trimEnd()}\n`);
            resolve({ name, secs, ok: code === 0 });
        });
    });
}

// pool runs names, n at a time
async function pool(names, n) {
    const queue = [...names];
    const results = [];
    const worker = async () => {
        while (queue.length) results.push(await run(queue.shift()));
    };
    await Promise.all(Array.from({ length: Math.min(n, queue.length) }, worker));
    return results;
}

const start = Date.now();
const results = [
    ...(await pool(scripts.filter((s) => !alone.includes(s)), jobs)),
    ...(await pool(scripts.filter((s) => alone.includes(s)), 1)),
];
const failed = results.filter((r) => !r.ok).map((r) => r.name);
const slowest = [...results].sort((a, b) => b.secs - a.secs).slice(0, 3).map((r) => `${r.name} ${r.secs}s`);
console.log(`${results.length - failed.length}/${results.length} scripts passed in ${Math.round((Date.now() - start) / 1000)}s; slowest: ${slowest.join(", ")}`);
if (failed.length) console.log(`failed: ${failed.join(" ")}`);
process.exit(failed.length ? 1 : 0);
