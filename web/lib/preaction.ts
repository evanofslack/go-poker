import { PlayerView, View } from "../interfaces/index";

// Pre is an action picked before your turn. It only lasts the street it was
// picked on, and a call is for one amount, so a raise clears it.
export type Pre = {
    kind: "checkFold" | "check" | "fold" | "call";
    amount: number; // to call
    street: string; // from preStreet
};

export type Act = "check" | "fold" | "call";

export function preStreet(view: View) {
    return `${view.hand}-${view.street}`;
}

// owed is what a player needs to call now, capped at their stack
export function owed(view: View, p: PlayerView) {
    const maxBet = Math.max(...view.players.map((q) => q.bet));
    return Math.min(maxBet - p.bet, p.stack);
}

// preOptions are the picks offered, fold-ish first like the turn buttons
export function preOptions(owe: number, stack: number): [Pre["kind"], string][] {
    if (owe === 0) {
        return [
            ["checkFold", "check/fold"],
            ["check", "check"],
        ];
    }
    return [
        ["fold", "fold"],
        ["call", owe === stack ? `all in ${owe}` : `call ${owe}`],
    ];
}

// resolvePre is what a pick does when owing owe, or null once it no longer applies
export function resolvePre(pre: Pre, owe: number): Act | null {
    switch (pre.kind) {
        case "checkFold":
            return owe === 0 ? "check" : "fold";
        case "check":
            return owe === 0 ? "check" : null;
        case "fold":
            return "fold";
        case "call":
            return owe === pre.amount ? "call" : null;
    }
}
