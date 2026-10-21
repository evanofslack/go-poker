export type Message = {
    name: string;
    message: string;
    timestamp: string;
};

export type Log = {
    message: string;
    timestamp: string;
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
    clientID: string | null;
    table: string | null;
    game: Game | null;
    error: string | null;
    connection: Connection;
    toasts: Toast[];
};

export type Card = string;

export type Player = {
    username: string;
    uuid: string;
    position: number;
    seatID: number;
    ready: boolean;
    in: boolean;
    called: boolean;
    left: boolean;
    totalBuyIn: number;
    stack: number;
    bet: number;
    totalBet: number;
    cards: Card[];
};

export type Game = {
    running: boolean;
    dealer: number;
    action: number;
    utg: number;
    sb: number;
    bb: number;
    communityCards: Card[];
    stage: number;
    betting: boolean;
    config: Config;
    players: Player[];
    pots: Pot[];
    minRaise: number;
    readyCount: number;
    host: string;
    turnEndsIn: number; // ms until the player to act times out, 0 when not betting
    sitOutNext: boolean; // you asked to sit out once this hand ends
    seats: number; // seats at the table
};

export type Config = {
    maxBuy: number;
    bb: number;
    sb: number;
};

export type Pot = {
    topShare: number;
    amount: number;
    eligiblePlayerNums: number[];
    winningPlayerNums: number[];
    winningHand: Card[];
    winningScore: number;
};
