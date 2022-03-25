export type Message = {
    name: string;
    message: string;
}

export type AppState = {
    messages: Message[]
    username: string
}