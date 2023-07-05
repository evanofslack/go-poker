export type Message = {
    name: string;
    message: string;
    timestamp: string;
};

export type Log = {
    message: string;
    timestamp: string;
};

export type AppState = {
    messages: Message[];
    logs: Log[];
    username: string | null;
    clientID: string | null;
    table: string | null;
    game: Game | null;
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
};

export type Config = {
    maxBuyIn: number;
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
