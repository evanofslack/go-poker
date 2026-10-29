import { HandStats, LedgerRow, PlayerTotals } from "../interfaces/index";

// the chart has 8 series colors; anyone past them is drawn in gray
export const seriesColors = 8;

export type PlayerStats = {
    name: string;
    played: number; // hands dealt in
    won: number; // hands won, splits included
    biggestPot: number;
    net: number; // from the ledger, so buy-ins don't count
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

// chartValue is what the chart plots for name after hand h: their net, or their
// stack for hands saved before nets were. undefined if they weren't seated.
export type chartValue = (h: HandStats, name: string) => number | undefined;

// chartValues picks what the chart plots: nets when every hand has them
export function chartValues(hands: HandStats[]): { net: boolean; value: chartValue } {
    if (hands.every((h) => h.net)) return { net: true, value: (h, n) => h.net[n] };
    return { net: false, value: (h, n) => h.stacks[n] };
}

// niceTicks spans lo to hi in a few round steps: 1, 2 or 5 times a power of ten
export function niceTicks(lo: number, hi: number): number[] {
    const raw = Math.max(hi - lo, 10) / 4;
    const mag = 10 ** Math.floor(Math.log10(raw));
    const step = [1, 2, 5, 10].map((m) => m * mag).find((s) => s >= raw)!;
    const ticks = [];
    for (let t = Math.floor(lo / step) * step; ; t += step) {
        ticks.push(t);
        if (t >= hi) break;
    }
    return ticks;
}

// playerStats totals each player's hands, biggest first by net
export function playerStats(hands: HandStats[], ledger: LedgerRow[]): PlayerStats[] {
    const last = hands[hands.length - 1];
    const nets = new Map(ledger.map((r) => [r.name, r.net]));
    const stats = names(hands).map((name) => {
        const s: PlayerStats = {
            name,
            played: 0,
            won: 0,
            biggestPot: 0,
            net: nets.get(name) ?? last?.net?.[name] ?? 0,
            stack: last?.stacks[name] ?? null,
        };
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
    return stats.sort((a, b) => b.net - a.net);
}

// fewHands is how many hands a player needs before their style stats settle;
// under it they show, marked as rough
export const fewHands = 10;

export type StyleStats = {
    name: string;
    hands: number;
    low: boolean; // too few hands to go by
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
            low: t.hands < fewHands,
            // bomb pots have no preflop to choose in
            vpip: pct(t.vpip, t.hands - t.bombs),
            pfr: pct(t.pfr, t.hands - t.bombs),
            threeBet: pct(t.threeBets, t.threeBetSpots),
            af: t.calls > 0 ? (t.aggressive / t.calls).toFixed(1) : t.aggressive > 0 ? "∞" : "–",
            wtsd: pct(t.showdowns, t.sawFlop),
            wsd: pct(t.showdownWins, t.showdowns),
            bb100: t.hands > 0 && bb > 0 ? ((100 * t.net) / bb / t.hands).toFixed(0) : "–",
            bestHand: t.bestHand ?? "",
        };
    });
}

// styleColumns label the play style table, with what each one means
export const styleColumns = [
    ["VPIP", "voluntarily put money in: hands they called or raised before the flop"],
    ["PFR", "preflop raise: hands they raised before the flop"],
    ["3B", "3-bet: re-raised a raise before the flop, when they had the chance"],
    ["AF", "aggression factor: bets and raises for each call"],
    ["WTSD", "went to showdown, of the hands they saw a flop in"],
    ["W$SD", "won at showdown, of the showdowns they went to"],
    ["bb/100", "big blinds won per 100 hands"],
];
