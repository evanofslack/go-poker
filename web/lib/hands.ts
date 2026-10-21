import { Game, Pot } from "../interfaces/index";

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

// potLabel describes who won a pot, like "side pot 40 · bob, a flush, ace high"
export function potLabel(game: Game, pot: Pot, index: number) {
    const name = game.pots.length === 1 ? "pot" : index === 0 ? "main pot" : "side pot";
    const names = pot.winningPlayerNums.map((n) => game.players[n]?.username ?? "?");
    const who = (names.length > 1 ? "split " : "") + names.join(", ");
    return `${name} ${pot.amount} · ${who}${pot.hand ? ", " + pot.hand : ""}`;
}
