import { Game } from "../interfaces/index";

export type Betting = ReturnType<typeof bettingInfo>;

// Betting numbers for the player to act. "To" amounts are the player's total bet for the
// street, which is what the raise input shows; the server takes the chips added on top.
export function bettingInfo(game: Game) {
    const player = game.players[game.action];
    const maxBet = Math.max(...game.players.map((p) => p.bet));
    const toCall = Math.min(maxBet - player.bet, player.stack);
    const maxTo = player.bet + player.stack;
    return {
        player,
        maxBet,
        toCall,
        maxTo,
        minTo: Math.min(maxBet + game.minRaise, maxTo),
        // every chip committed this hand, including this street's bets
        pot: game.players.reduce((sum, p) => sum + p.totalBet, 0),
        opening: maxBet === 0,
        canRaise: player.stack > toCall,
        // a short stack can only shove, and a short all-in doesn't reopen betting
        fullRaise: !player.called && maxTo > maxBet + game.minRaise,
    };
}

// potRaiseTo is a raise of a fraction of the pot after calling
export function potRaiseTo(b: Betting, fraction: number) {
    return b.maxBet + Math.round((b.pot + b.toCall) * fraction);
}

export function clampRaise(b: Betting, to: number) {
    return Math.min(Math.max(to, b.minTo), b.maxTo);
}
