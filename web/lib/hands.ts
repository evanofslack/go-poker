import { Game, Pot } from "../interfaces/index";

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

// potLabel describes who won a pot, like "side pot 40 · bob, a flush"
export function potLabel(game: Game, pot: Pot, index: number) {
    const name = game.pots.length === 1 ? "pot" : index === 0 ? "main pot" : "side pot";
    const names = pot.winningPlayerNums.map((n) => game.players[n]?.username ?? "?");
    const hand = handName(pot.winningScore);
    const who = (names.length > 1 ? "split " : "") + names.join(", ");
    return `${name} ${pot.amount} · ${who}${hand ? ", " + hand : ""}`;
}
