import { View } from "../interfaces/index";

export type Betting = ReturnType<typeof bettingInfo>;

// Betting numbers for the player to act, from what the server says they may do.
// "To" amounts are the player's total bet for the street, which is what the
// raise input shows and the server takes.
export function bettingInfo(view: View) {
    const player = view.players.find((p) => p.seat === view.toAct)!;
    const { call, raiseMin, raiseMax } = view.legal;
    const maxBet = Math.max(...view.players.map((p) => p.bet));
    return {
        player,
        maxBet,
        toCall: call,
        maxTo: raiseMax,
        minTo: Math.min(raiseMin, raiseMax),
        // every chip committed this hand, including this street's bets
        pot: view.players.reduce((sum, p) => sum + p.committed, 0),
        opening: maxBet === 0,
        canRaise: raiseMax > 0,
        // a short stack can only shove, and a short all-in doesn't reopen betting
        fullRaise: raiseMax > raiseMin,
    };
}

// potRaiseTo is a raise of a fraction of the pot after calling
export function potRaiseTo(b: Betting, fraction: number) {
    return b.maxBet + Math.round((b.pot + b.toCall) * fraction);
}

export function clampRaise(b: Betting, to: number) {
    return Math.min(Math.max(to, b.minTo), b.maxTo);
}
