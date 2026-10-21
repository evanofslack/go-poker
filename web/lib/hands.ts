import { Card, Game, Pot } from "../interfaces/index";

const preDeal = 1;

// handName mirrors the server's handName. Scores are Cactus Kev ranks, lower is better,
// and a pot won without a showdown scores 8000 and has no name.
export function handName(score: number) {
    if (score <= 0 || score > 7462) return "";
    if (score <= 10) return "a straight flush";
    if (score <= 166) return "four of a kind";
    if (score <= 322) return "a full house";
    if (score <= 1599) return "a flush";
    if (score <= 1609) return "a straight";
    if (score <= 2467) return "three of a kind";
    if (score <= 3325) return "two pair";
    if (score <= 6185) return "a pair";
    return "high card";
}

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
    const pots = settledPots(game).filter((p) => handName(p.winningScore));
    return new Set(pots.flatMap((p) => (p.winningHand ?? []).map(String)));
}

// winningHand names the best hand the player at position won with at showdown,
// like "flush", or is empty if they won without showing
export function winningHand(game: Game, position: number) {
    const scores = settledPots(game)
        .filter((p) => p.winningPlayerNums.includes(position))
        .map((p) => p.winningScore);
    return handName(Math.min(...scores)).replace(/^an? /, "");
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
