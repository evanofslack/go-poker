import { Card, Pot, View } from "../interfaces/index";

// settledPots are the pots awarded in the last hand, shown until the next deal
export function settledPots(view: View): Pot[] {
    if (view.street !== "") return [];
    return view.pots.filter((p) => p.winners.length);
}

export function isWinner(view: View, seat: number) {
    return settledPots(view).some((p) => p.winners.includes(seat));
}

// winningCards are the cards that made a winning hand at showdown
export function winningCards(view: View) {
    return new Set(settledPots(view).flatMap((p) => p.runs.flatMap((r) => r.cards)));
}

// winningHand names the hand the player in seat won with at showdown, like
// "two pair, kings and sevens", or is empty if they won without showing
export function winningHand(view: View, seat: number) {
    const runs = settledPots(view).flatMap((p) => p.runs);
    return runs.find((r) => r.hand && r.winners.includes(seat))?.hand ?? "";
}

// winnings are the chips the player in seat took from the pots last hand: a
// share of each run they won at showdown, or the pot when everyone else folded
export function winnings(view: View, seat: number) {
    type share = { amount: number; winners: number[] };
    const shares = settledPots(view).flatMap((p): share[] => (p.runs.length ? p.runs : [p]));
    return shares
        .filter((s) => s.winners.includes(seat))
        .reduce((sum, s) => sum + Math.floor(s.amount / s.winners.length), 0);
}

// hasStraight says whether ranks (0 for a deuce to 12 for an ace) hold five in a row,
// counting the ace low too
function hasStraight(ranks: Set<number>) {
    const run = [12, ...Array.from({ length: 13 }, (_, i) => i)];
    let len = 0;
    for (const r of run) {
        len = ranks.has(r) ? len + 1 : 0;
        if (len === 5) return true;
    }
    return false;
}

// madeHand names the best hand in cards, like "two pair". Cards are the server's
// ints, with the rank in bits 8-11 and the suit in bits 12-15.
export function madeHand(cards: Card[]) {
    const cs = cards.filter((c) => c);
    if (cs.length < 2) return "";
    const rank = (c: number) => (c >> 8) & 0xf;
    const bySuit = new Map<number, Set<number>>();
    const counts = new Map<number, number>();
    for (const c of cs) {
        const suit = bySuit.get(c & 0xf000) ?? new Set();
        bySuit.set(c & 0xf000, suit.add(rank(c)));
        counts.set(rank(c), (counts.get(rank(c)) ?? 0) + 1);
    }
    const flush = Array.from(bySuit.values()).find((s) => s.size >= 5);
    const n = Array.from(counts.values()).sort((a, b) => b - a);
    if (flush && hasStraight(flush)) return "straight flush";
    if (n[0] === 4) return "four of a kind";
    if (n[0] === 3 && n[1] >= 2) return "full house";
    if (flush) return "flush";
    if (hasStraight(new Set(counts.keys()))) return "straight";
    if (n[0] === 3) return "three of a kind";
    if (n[0] === 2 && n[1] === 2) return "two pair";
    if (n[0] === 2) return "pair";
    return "high card";
}

// blind labels the small and big blind seats, only preflop when they matter
export function blind(view: View, seat: number) {
    if (view.street !== "preflop") return "";
    if (view.sb === seat) return "SB";
    if (view.bb === seat) return "BB";
    if (view.straddles.includes(seat)) return "STR";
    return "";
}
