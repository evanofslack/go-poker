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
    password: string | null; // the table's, for invite links
    game: Game | null;
    error: string | null;
    needPassword: boolean; // the last join was refused for a missing or wrong password
    connection: Connection;
    toasts: Toast[];
    hands: HandStats[]; // the last hands played, oldest first
};

// HandStats is a snapshot taken as a hand ends.
export type HandStats = {
    hand: number; // counts from 1
    stacks: Record<string, number>; // everyone seated, after the pots are paid
    played: string[]; // names dealt in
    won: Record<string, number>; // pots each winner took, their own chips included
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
    host: string; // name of the host
    turnEndsIn: number; // ms until the player to act times out, 0 when not betting
    sitOutNext: boolean; // you asked to sit out once this hand ends
    settings: Settings; // chosen by the host when creating the table
    hasPassword: boolean;
    paused: boolean; // no new hands are dealt
    ending: boolean; // the session ends after this hand
    ended: boolean; // the session is over and the ledger final
    closesIn: number; // ms until an ended table closes
    buyRequests: BuyRequest[]; // waiting for the host, oldest first
    watchers: string[]; // names of the clients at the table without a seat
    disconnected: string[]; // names of seated players who dropped and may still reconnect
    bots: Record<string, string>; // level of each bot, by uuid
    ledger: Ledger;
};

// BuyRequest is a rebuy, or a returning player sitting back down, waiting for
// the host. seat is 0 for a rebuy.
export type BuyRequest = {
    name: string;
    amount: number;
    seat: number;
};

// Settings are fixed once the table exists. Buy-ins are in chips, times in ms.
export type Settings = {
    seats: number;
    sb: number;
    bb: number;
    minBuy: number;
    maxBuy: number;
    turnTime: number; // 0 for no turn timer
    nextHandDelay: number;
    public: boolean;
    approveRebuys: boolean; // rebuys, and sitting back down, wait for the host
};

// LedgerRow is one name's chips in and out. Someone who leaves and comes back
// under the same name continues their row.
export type LedgerRow = {
    name: string;
    boughtIn: number;
    cashedOut: number;
    stack: number; // chips at the table now, counting this hand's bets
    net: number; // stack + cashedOut - boughtIn
};

export type Ledger = {
    rows: LedgerRow[];
    banker: string; // everyone settles up with the banker
    balance: number; // sum of nets, 0 when every chip is accounted for
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
