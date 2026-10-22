import { HandAction, HandRecord } from "../interfaces/index";

const streetNames = ["preflop", "flop", "turn", "river"] as const;

export type Street = {
    name: (typeof streetNames)[number];
    cards: string[]; // dealt on this street, like "Ks"
    actions: HandAction[];
};

// streets splits a hand's actions by street, with the board cards each one
// dealt. Streets run out after an all in show up with no actions.
export function streets(r: HandRecord): Street[] {
    const board = r.board ?? [];
    const dealt = [[], board.slice(0, 3), board.slice(3, 4), board.slice(4, 5)];
    return streetNames
        .map((name, i) => ({ name, cards: dealt[i], actions: r.actions.filter((a) => a.street === name) }))
        .filter((s) => s.name === "preflop" || s.cards.length > 0);
}

// actionText describes an action without its player, like "raises to 120"
export function actionText(a: HandAction): string {
    let text: string;
    switch (a.kind) {
        case "small blind":
        case "big blind":
            text = `posts ${a.kind} ${a.amount ?? 0}`;
            break;
        case "check":
            text = "checks";
            break;
        case "call":
            text = `calls ${a.amount}`;
            break;
        case "bet":
            text = `bets ${a.amount}`;
            break;
        case "raise":
            text = `raises to ${a.to}`;
            break;
        case "fold":
            text = "folds";
            break;
        default:
            text = a.kind;
    }
    if (a.allIn) text += ", all in";
    if (a.timeout) text += " (out of time)";
    return text;
}

// winnersText says who won a hand, like "bob wins 340" or "alice and bob split 80"
export function winnersText(r: HandRecord): string {
    const pots = r.pots ?? [];
    if (!pots.length) return "";
    const total = pots.reduce((sum, p) => sum + p.amount, 0);
    const names = Array.from(new Set(pots.flatMap((p) => p.winners)));
    if (pots.length === 1 && names.length > 1) return `${names.join(" and ")} split ${total}`;
    return `${names.join(", ")} ${names.length > 1 ? "win" : "wins"} ${total}`;
}

// netText writes chips won or lost with a sign, like "+120"
export function netText(net: number): string {
    return net > 0 ? `+${net}` : String(net);
}
