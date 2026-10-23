// The protocol types are generated from the go code by `just types`, into
// holdem.ts, table.ts and server.ts. This file adds what tygo can't: the event
// union, and the client's own state.
import type * as holdem from "./holdem";
import type * as table from "./table";

export type {
    BuyRequest,
    HandAction,
    HandPlayer,
    HandPot,
    HandRecord,
    HandStats,
    LedgerRow,
    LedgerView,
    PlayerView,
    Settings,
    Summary as TableSummary,
    Totals as PlayerTotals,
    View,
} from "./table";
export type { Card, Legal, Pot } from "./holdem";

// GameEvent is something that happened at the table, as the server sends it
// with a table update.
export type GameEvent =
    | { type: "hand-started"; data: holdem.HandStarted }
    | { type: "dealt"; data: holdem.Dealt }
    | { type: "acted"; data: holdem.Acted }
    | { type: "street-dealt"; data: holdem.StreetDealt }
    | { type: "returned"; data: holdem.Returned }
    | { type: "shown"; data: holdem.Shown }
    | { type: "pot-won"; data: holdem.PotWon }
    | { type: "hand-ended"; data: holdem.HandEnded }
    | { type: "hand-done"; data: table.HandDone }
    | { type: "log"; data: table.Logged };

export type Message = {
    name: string;
    message: string;
    time: number; // unix ms
};

export type Log = {
    id?: number; // from the server; errors shown in the log have none
    message: string;
    time: number; // unix ms
};

// connecting until the first open; reconnecting after a drop
export type Connection = "connecting" | "open" | "reconnecting";

export type Toast = {
    id: number;
    message: string;
    kind: "info" | "error";
};

export type AppState = {
    messages: Message[];
    logs: Log[];
    username: string | null;
    table: string | null;
    password: string | null; // the table's, for invite links
    view: table.View | null;
    error: string | null;
    needPassword: boolean; // the last join was refused for a missing or wrong password
    connection: Connection;
    toasts: Toast[];
    hands: table.HandStats[]; // the last hands played, oldest first
    totals: Record<string, table.Totals>; // by name, over every hand
    history: table.HandRecord[]; // hands as played, oldest first; older ones are fetched with getHands
    moreHistory: boolean; // the table keeps hands older than history's first
    tables: table.Summary[]; // public tables for the home page, best to join first
    shortcuts: boolean; // keyboard shortcuts for the betting controls
};
