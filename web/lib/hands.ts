import { Card, Game, Pot } from "../interfaces/index";

const preDeal = 1;

// settledPots are the pots awarded in the last hand, shown until the next deal
export function settledPots(game: Game): Pot[] {
    if (game.stage !== preDeal) return [];
    return game.pots.filter((p) => p.winningPlayerNums?.length);
}

export function isWinner(game: Game, position: number) {
    return settledPots(game).some((p) => p.winningPlayerNums.includes(position));
}

// winningCards are the cards that made a winning hand at showdown
export function winningCards(game: Game) {
    const pots = settledPots(game).filter((p) => p.hand);
    return new Set(pots.flatMap((p) => (p.winningHand ?? []).map(String)));
}

// winningHand names the hand the player at position won with at showdown, like
// "two pair, kings and sevens", or is empty if they won without showing
export function winningHand(game: Game, position: number) {
    const pot = settledPots(game).find((p) => p.hand && p.winningPlayerNums.includes(position));
    return pot?.hand ?? "";
}

// winnings are the chips the player at position took from the pots last hand
export function winnings(game: Game, position: number) {
    return settledPots(game)
        .filter((p) => p.winningPlayerNums.includes(position))
        .reduce((sum, p) => sum + Math.floor(p.amount / p.winningPlayerNums.length), 0);
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
    const cs = cards.filter((c) => c && c !== "?").map(Number);
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
