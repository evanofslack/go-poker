import { HandStats } from "../interfaces/index";

// the chart has 8 series colors; anyone past them is drawn in gray
export const seriesColors = 8;

export type PlayerStats = {
    name: string;
    played: number; // hands dealt in
    won: number; // hands won, splits included
    biggestPot: number;
    stack: number | null; // after the last hand, null if they've left
};

// names lists everyone in hands in the order they first show up, which fixes
// their color as others come and go
export function names(hands: HandStats[]): string[] {
    const seen = new Set<string>();
    for (const h of hands) {
        for (const n of h.played) seen.add(n);
        for (const n of Object.keys(h.stacks).sort()) seen.add(n);
    }
    return Array.from(seen);
}

// seriesColor is name's chart color, from its place in names
export function seriesColor(i: number): string {
    return i < seriesColors ? `var(--color-series-${i + 1})` : "var(--color-subtle)";
}

export function playerStats(hands: HandStats[]): PlayerStats[] {
    const last = hands[hands.length - 1];
    return names(hands).map((name) => {
        const s: PlayerStats = { name, played: 0, won: 0, biggestPot: 0, stack: last?.stacks[name] ?? null };
        for (const h of hands) {
            if (h.played.includes(name)) s.played++;
            const won = h.won[name];
            if (won) {
                s.won++;
                s.biggestPot = Math.max(s.biggestPot, won);
            }
        }
        return s;
    });
}
