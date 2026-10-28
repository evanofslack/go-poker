// Prefs are how one viewer likes the table: how it looks, what it plays, and
// their keys. They're kept in this browser only, under one localStorage key.
export type Prefs = {
    motion: "system" | "on" | "off"; // table animations; system follows reduced motion
    deck: "four" | "two"; // suit colors
    felt: "green" | "blue" | "red" | "gray";
    back: "red" | "blue" | "black"; // card backs
    shortcuts: boolean; // keyboard shortcuts for the betting controls
    muted: boolean; // every sound, from the header's button
};

export const defaultPrefs: Prefs = {
    motion: "system",
    deck: "four",
    felt: "green",
    back: "red",
    shortcuts: true,
    muted: false,
};

const key = "prefs";

// loadPrefs is the saved prefs over the defaults. The first time, it carries
// over the settings that were saved on their own before.
export function loadPrefs(): Prefs {
    try {
        const saved = localStorage.getItem(key);
        if (saved) return { ...defaultPrefs, ...JSON.parse(saved) };
        return {
            ...defaultPrefs,
            shortcuts: localStorage.getItem("shortcuts") !== "0",
            muted: localStorage.getItem("muted") === "1",
        };
    } catch {
        return defaultPrefs;
    }
}

// looks are the prefs styles/index.css reads as data attributes on <html>, as
// data-deck, data-felt and data-back. pages/_document.tsx sets them before the
// first paint; showLooks keeps them in step after.
const looks = ["deck", "felt", "back"] as const;

export function showLooks(prefs: Prefs) {
    for (const look of looks) document.documentElement.dataset[look] = prefs[look];
}

// looksScript is showLooks for the page's <head>, before React loads
export const looksScript = `try {
    const p = { ...${JSON.stringify(Object.fromEntries(looks.map((l) => [l, defaultPrefs[l]])))}, ...JSON.parse(localStorage.getItem("${key}") || "{}") };
    for (const l of ${JSON.stringify(looks)}) document.documentElement.dataset[l] = p[l];
} catch {}`;

export function savePrefs(prefs: Prefs) {
    try {
        localStorage.setItem(key, JSON.stringify(prefs));
    } catch {}
}
