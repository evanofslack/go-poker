// Shared helpers for the e2e scripts.
import { chromium } from "playwright-core";

export const URL = process.env.URL || "http://localhost:8099";

// launch uses CHROMIUM_PATH if set, else playwright's own chromium
export function launch() {
    const exe = process.env.CHROMIUM_PATH;
    return chromium.launch(exe ? { executablePath: exe } : {});
}

let passed = 0;
let failed = 0;

export function check(name, ok, detail = "") {
    if (ok) passed++;
    else failed++;
    console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " — " + detail : ""}`);
}

// confirmCreate creates the table from the settings dialog, with its defaults
export async function confirmCreate(page) {
    await page.click('button:text-is("create table")');
}

// shot saves a screenshot when SHOTS names a directory
export async function shot(page, name) {
    if (process.env.SHOTS) await page.screenshot({ path: `${process.env.SHOTS}/${name}.png` });
}

// finish prints the tally and exits nonzero on any failure
export function finish() {
    console.log(`\n${passed}/${passed + failed} passed`);
    process.exit(failed ? 1 : 0);
}
