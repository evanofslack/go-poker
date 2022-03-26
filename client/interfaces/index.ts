export type Message = {
    name: string;
    message: string;
    timestamp: string;
};

export type AppState = {
    messages: Message[];
    username: string;
};

export type Event = {
    action: string;
    params: any;
};

export type Card = string;

export type Player = {
    username: string;
    seatID: number;
    totalBuyIn: number;
    stack: number;
    ready: boolean;
    in: boolean;
    called: boolean;
    left: boolean;
    bet: number;
    totalBet: number;
    cards: Card[];
};

export type Game = {
    dealer: Player;
    action: Player;
    utg: Player;
    sb: Player;
    bb: Player;
    communityCards: Card[];
    stage: number;
    betting: boolean;
    config: Config;
    players: Player[];
};

export type Config = {
    maxBuyIn: number;
    bb: number;
    sb: number;
};

export type Pot = {
    topShare: number;
    amount: number;
    eligiblePlayers: Player[];
    winningPlayers: Player[];
    winningHand: Card[];
    winningScore: number;
};
