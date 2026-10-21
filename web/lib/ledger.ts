import type { Dispatch } from "react";
import { Game, LedgerRow } from "../interfaces/index";

export type PlayerStatus = "playing" | "sitting out" | "out of chips" | "disconnected" | "watching" | "left";

// PlayerLine is one person in the players panel: a ledger row, or a spectator
// who never bought in
export type PlayerLine = {
    name: string;
    status: PlayerStatus;
    seat: number | null;
    row: LedgerRow | null;
    bot: string | null; // the bot's level, if it's a bot
};

// playerLines lists everyone with a ledger row, in the order they first bought
// in, then spectators who never did
export function playerLines(game: Game): PlayerLine[] {
    const status = (name: string): [PlayerStatus, number | null] => {
        const p = game.players.find((p) => p.username === name);
        if (p) {
            if (game.disconnected.includes(name)) return ["disconnected", p.seatID];
            if (p.ready) return ["playing", p.seatID];
            return [p.stack > 0 ? "sitting out" : "out of chips", p.seatID];
        }
        return [game.watchers.includes(name) ? "watching" : "left", null];
    };
    const bot = (name: string) => {
        const p = game.players.find((p) => p.username === name);
        return (p && game.bots[p.uuid]) || null;
    };
    const lines: PlayerLine[] = game.ledger.rows.map((row) => {
        const [s, seat] = status(row.name);
        return { name: row.name, status: s, seat, row, bot: bot(row.name) };
    });
    for (const name of game.watchers) {
        if (!lines.some((l) => l.name === name)) {
            lines.push({ name, status: "watching", seat: null, row: null, bot: null });
        }
    }
    return lines;
}

// settleLine is what name does to settle up with the banker
export function settleLine(name: string, net: number, banker: string) {
    if (!banker) return "";
    if (name === banker) return "banker";
    if (net > 0) return `request ${net} from ${banker}`;
    if (net < 0) return `send ${-net} to ${banker}`;
    return "even";
}

// signed shows a net with its sign
export function signed(n: number) {
    return n > 0 ? `+${n}` : String(n);
}

// ledgerText is the ledger as plain text, for pasting into a group chat
export function ledgerText(table: string, game: Game) {
    const { rows, banker, balance } = game.ledger;
    const width = Math.max(0, ...rows.map((r) => r.name.length));
    const lines = rows.map(
        (r) => `${r.name.padEnd(width)}  ${signed(r.net).padStart(7)}  ${settleLine(r.name, r.net, banker)}`
    );
    const check = balance === 0 ? "balanced" : `off by ${balance}`;
    return [`${table} ledger, in chips (banker: ${banker || "none"})`, ...lines, check].join("\n");
}

// copyLedger copies the ledger text and says whether it worked
export async function copyLedger(table: string, game: Game, dispatch: Dispatch<any>) {
    try {
        await navigator.clipboard.writeText(ledgerText(table, game));
        dispatch({ type: "addToast", payload: { message: "ledger copied", kind: "info" } });
    } catch {
        dispatch({ type: "addToast", payload: { message: "couldn't copy the ledger", kind: "error" } });
    }
}
