import { Pot, View } from "../interfaces/index";

// settledPots are the pots awarded in the last hand, shown until the next deal
export function settledPots(view: View): Pot[] {
    if (view.street !== "") return [];
    return view.pots.filter((p) => p.winners.length);
}

export function isWinner(view: View, seat: number) {
    return settledPots(view).some((p) => p.winners.includes(seat));
}

// winningCards are the cards that made a winning hand at showdown, on every
// board or only the given run's
export function winningCards(view: View, run?: number) {
    return new Set(
        settledPots(view).flatMap((p) => p.runs.filter((_, i) => run === undefined || i === run).flatMap((r) => r.cards))
    );
}

// runWinners names who won each run of the main pot, like "alice" or "alice
// and bob", once the board ran out more than once
export function runWinners(view: View): { names: string; hand: string }[] {
    const main = settledPots(view)[0];
    if (!main || main.runs.length < 2) return [];
    const name = (seat: number) => view.players.find((p) => p.seat === seat)?.name ?? "";
    return main.runs.map((r) => ({ names: r.winners.map(name).join(" and "), hand: r.hand }));
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

// blind labels the small and big blind seats, only preflop when they matter
export function blind(view: View, seat: number) {
    if (view.street !== "preflop") return "";
    if (view.sb === seat) return "SB";
    if (view.bb === seat) return "BB";
    if (view.straddles.includes(seat)) return "STR";
    return "";
}
