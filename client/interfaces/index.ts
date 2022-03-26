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
