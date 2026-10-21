export type Message = {
    name: string;
    message: string;
    timestamp: string;
};

export type Log = {
    id?: string; // from the server; errors shown in the log have none
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
    totals: Record<string, PlayerTotals>; // by name, over every hand
    history: HandRecord[]; // hands as played, oldest first; older ones are fetched with getHands
    moreHistory: boolean; // the table keeps hands older than history's first
    tables: TableSummary[]; // public tables for the home page, best to join first
};

// TableSummary is what the home page shows of a public table
export type TableSummary = {
    name: string;
    sb: number;
    bb: number;
    minBuy: number;
    maxBuy: number;
    seats: number;
    players: number; // seated, bots included
    bots: number;
    watchers: number; // connected without a seat
    running: boolean;
    paused: boolean;
    hasPassword: boolean;
};

// HandStats is a snapshot taken as a hand ends.
export type HandStats = {
    hand: number; // counts from 1
    stacks: Record<string, number>; // everyone seated, after the pots are paid
    played: string[]; // names dealt in
    won: Record<string, number>; // pots each winner took, their own chips included
};

// PlayerTotals add up how someone has played every hand at the table
export type PlayerTotals = {
    hands: number; // dealt in
    vpip: number; // hands they chose to put money in preflop
    pfr: number; // hands they raised preflop
    threeBetSpots: number; // hands they acted preflop facing one raise
    threeBets: number; // and re-raised
    aggressive: number; // bets and raises
    calls: number;
    sawFlop: number; // hands still in when the flop came
    showdowns: number; // hands they showed
    showdownWins: number; // and won a pot
    net: number; // chips won or lost
    bestHand?: string;
};

// HandRecord is one hand as it was played. Other players' cards are only
// there if they were shown at showdown.
export type HandRecord = {
    hand: number; // counts from 1
    started: number; // unix ms
    sb: number;
    bb: number;
    dealer: string;
    players: HandPlayer[]; // dealt in, in seat order
    actions: HandAction[]; // blinds first
    board: Card[] | null;
    pots: HandPot[] | null;
};

export type HandPlayer = {
    name: string;
    seat: number;
    stack: number; // before the blinds
    cards?: Card[]; // yours, or shown at showdown
    shown: boolean;
    hand?: string; // their best hand, if shown
    net: number; // chips won or lost
};

export type HandAction = {
    street: "preflop" | "flop" | "turn" | "river";
    name: string;
    kind: "small blind" | "big blind" | "check" | "call" | "bet" | "raise" | "fold";
    amount?: number; // chips put in
    to?: number; // their bet on this street after
    allIn?: boolean;
    timeout?: boolean; // the turn timer acted for them
};

export type HandPot = {
    amount: number;
    winners: string[];
    hand?: string; // empty without a showdown
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
    acts: Record<string, string>; // each player's action this street, and folds this hand, by uuid
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
    hand: string; // named by the server, like "two pair, kings and sevens"; empty without a showdown
};
