import { HandStats, PlayerTotals } from "../interfaces/index";

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

// fewHands is how many hands a player needs before their style stats mean much
export const fewHands = 20;

export type StyleStats = {
    name: string;
    hands: number;
    vpip: string;
    pfr: string;
    threeBet: string;
    af: string; // aggression factor
    wtsd: string; // went to showdown, of hands that saw the flop
    wsd: string; // won at showdown
    bb100: string; // big blinds won per 100 hands
    bestHand: string;
};

const pct = (n: number, of: number) => (of > 0 ? `${Math.round((100 * n) / of)}%` : "–");

// styleStats works out how each player plays from their totals, in order
export function styleStats(totals: Record<string, PlayerTotals>, order: string[], bb: number): StyleStats[] {
    const all = [...order.filter((n) => totals[n]), ...Object.keys(totals).filter((n) => !order.includes(n)).sort()];
    return all.map((name) => {
        const t = totals[name];
        return {
            name,
            hands: t.hands,
            vpip: pct(t.vpip, t.hands),
            pfr: pct(t.pfr, t.hands),
            threeBet: pct(t.threeBets, t.threeBetSpots),
            af: t.calls > 0 ? (t.aggressive / t.calls).toFixed(1) : t.aggressive > 0 ? "∞" : "–",
            wtsd: pct(t.showdowns, t.sawFlop),
            wsd: pct(t.showdownWins, t.showdowns),
            bb100: t.hands > 0 && bb > 0 ? ((100 * t.net) / bb / t.hands).toFixed(0) : "–",
            bestHand: t.bestHand ?? "",
        };
    });
}
