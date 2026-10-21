import { Settings } from "../interfaces/index";

// createTable makes a new table; the server uses its defaults without settings
export function createTable(
    socket: WebSocket,
    tablename: string,
    username: string,
    settings?: Settings,
    password?: string
) {
    socket.send(
        JSON.stringify({
            action: "create-table",
            tablename: tablename,
            username: username,
            settings: settings,
            password: password,
        })
    );
}

export function joinTable(
    socket: WebSocket,
    tablename: string,
    username: string,
    token?: string,
    password?: string
) {
    socket.send(
        JSON.stringify({
            action: "join-table",
            tablename: tablename,
            username: username,
            token: token,
            password: password,
        })
    );
}

export function sendMessage(socket: WebSocket, username: string, message: string) {
    socket.send(
        JSON.stringify({
            action: "send-message",
            username: username,
            message: message,
        })
    );
}

export function takeSeat(socket: WebSocket, username: string, seatID: number, buyIn: number) {
    socket.send(
        JSON.stringify({
            action: "take-seat",
            username: username,
            seatID: seatID,
            buyIn: buyIn,
        })
    );
}

export function startGame(socket: WebSocket) {
    socket.send(
        JSON.stringify({
            action: "start-game",
        })
    );
}

export function playerCall(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-call",
        })
    );
}

export function playerCheck(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-check",
        })
    );
}

export function playerRaise(socket: WebSocket, amount: number) {
    socket?.send(
        JSON.stringify({
            action: "player-raise",
            amount: amount,
        })
    );
}

export function playerFold(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-fold",
        })
    );
}

// leaveTable leaves the table, giving up any seat
export function leaveTable(socket: WebSocket, tablename: string) {
    socket?.send(
        JSON.stringify({
            action: "leave-table",
            tablename: tablename,
        })
    );
}

// leaveSeat gives up the seat but stays at the table as a spectator
export function leaveSeat(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "leave-seat",
        })
    );
}

// sitOut toggles sitting out: right away between hands, or once the current hand ends
export function sitOut(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "sit-out",
        })
    );
}

export function rebuy(socket: WebSocket, amount: number) {
    socket?.send(
        JSON.stringify({
            action: "rebuy",
            amount: amount,
        })
    );
}

// addBot has the host sit a bot down; level is "easy" or "medium"
export function addBot(
    socket: WebSocket,
    seatID: number,
    buyIn: number,
    name: string,
    level: string
) {
    socket?.send(
        JSON.stringify({
            action: "add-bot",
            seatID: seatID,
            buyIn: buyIn,
            name: name,
            level: level,
        })
    );
}

// host only: actions on the player called name
export type HostAction = "make-host" | "set-banker" | "move-to-spectator" | "remove-player" | "sit-out-player";

export function hostAction(socket: WebSocket, action: HostAction, name: string) {
    socket?.send(
        JSON.stringify({
            action: action,
            name: name,
        })
    );
}

// removeBot has the host take the bot in a seat out of the game
export function removeBot(socket: WebSocket, seatID: number) {
    socket?.send(
        JSON.stringify({
            action: "remove-bot",
            seatID: seatID,
        })
    );
}

// adjustChips adds amount to name's stack, or takes it away when negative (host only)
export function adjustChips(socket: WebSocket, name: string, amount: number, reason: string) {
    socket?.send(
        JSON.stringify({
            action: "adjust-chips",
            name: name,
            amount: amount,
            reason: reason,
        })
    );
}
