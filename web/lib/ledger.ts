import type { Dispatch } from "react";
import { LedgerRow, View } from "../interfaces/index";

export type PlayerStatus = "playing" | "sitting out" | "out of chips" | "disconnected" | "watching" | "left";

// PlayerLine is one person in the table panel: a ledger row, or a spectator
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
export function playerLines(view: View): PlayerLine[] {
    const line = (name: string): PlayerLine => {
        const p = view.players.find((p) => p.name === name && !p.leaving);
        let status: PlayerStatus = view.watchers.includes(name) ? "watching" : "left";
        if (p) {
            if (p.disconnected) status = "disconnected";
            else if (!p.sittingOut) status = "playing";
            else status = p.stack > 0 ? "sitting out" : "out of chips";
        }
        return { name, status, seat: p?.seat ?? null, row: null, bot: p?.bot || null };
    };
    const lines: PlayerLine[] = view.ledger.rows.map((row) => ({ ...line(row.name), row }));
    for (const name of view.watchers) {
        if (!lines.some((l) => l.name === name)) lines.push(line(name));
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
export function ledgerText(table: string, view: View) {
    const { rows, banker, balance } = view.ledger;
    const width = Math.max(0, ...rows.map((r) => r.name.length));
    const lines = rows.map(
        (r) => `${r.name.padEnd(width)}  ${signed(r.net).padStart(7)}  ${settleLine(r.name, r.net, banker)}`
    );
    const check = balance === 0 ? "balanced" : `off by ${balance}`;
    return [`${table} ledger, in chips (banker: ${banker || "none"})`, ...lines, check].join("\n");
}

// copyLedger copies the ledger text and says whether it worked
export async function copyLedger(table: string, view: View, dispatch: Dispatch<any>) {
    try {
        await navigator.clipboard.writeText(ledgerText(table, view));
        dispatch({ type: "addToast", payload: { message: "ledger copied", kind: "info" } });
    } catch {
        dispatch({ type: "addToast", payload: { message: "couldn't copy the ledger", kind: "error" } });
    }
}
