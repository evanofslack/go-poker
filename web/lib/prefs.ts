// Prefs are how one viewer likes the table: how it looks, what it plays, and
// their keys. They're kept in this browser only, under one localStorage key.
export type Prefs = {
    motion: "system" | "on" | "off"; // table animations; system follows reduced motion
    shortcuts: boolean; // keyboard shortcuts for the betting controls
    muted: boolean; // every sound, from the header's button
};

export const defaultPrefs: Prefs = {
    motion: "system",
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

export function savePrefs(prefs: Prefs) {
    try {
        localStorage.setItem(key, JSON.stringify(prefs));
    } catch {}
}
