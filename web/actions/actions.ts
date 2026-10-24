import { Settings } from "../interfaces/index";
import type { Inbound } from "../interfaces/server";

function send(socket: WebSocket | null, msg: Inbound) {
    socket?.send(JSON.stringify(msg));
}

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

export function sendMessage(socket: WebSocket, message: string) {
    send(socket, { action: "send-message", message });
}

export function takeSeat(socket: WebSocket, seat: number, buyIn: number) {
    send(socket, { action: "take-seat", seat, amount: buyIn });
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

// playerRaise bets or raises to to, the player's whole bet this street
export function playerRaise(socket: WebSocket, to: number) {
    send(socket, { action: "player-raise", to });
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

// showCards turns up your cards from the last hand
export function showCards(socket: WebSocket) {
    send(socket, { action: "show-cards" });
}

// duckHunt turns up the rest of the board after a hand that ended early
export function duckHunt(socket: WebSocket) {
    send(socket, { action: "duck-hunt" });
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
export function addBot(socket: WebSocket, seat: number, buyIn: number, name: string, level: string) {
    send(socket, { action: "add-bot", seat, amount: buyIn, name, level });
}

// host only: actions on the player called name
export type HostAction =
    | "make-host"
    | "set-banker"
    | "move-to-spectator"
    | "remove-player"
    | "sit-out-player"
    | "approve-buy-in"
    | "deny-buy-in";

export function hostAction(socket: WebSocket, action: HostAction, name: string) {
    socket?.send(
        JSON.stringify({
            action: action,
            name: name,
        })
    );
}

// removeBot has the host take the bot in a seat out of the game
export function removeBot(socket: WebSocket, seat: number) {
    send(socket, { action: "remove-bot", seat });
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

// host only: stop dealing new hands, start again, or end the session for good
export function sessionAction(socket: WebSocket, action: "pause-game" | "resume-game" | "end-session") {
    socket?.send(
        JSON.stringify({
            action: action,
        })
    );
}

// getHands asks for the hands played before hand number before, answered with hand-history
export function getHands(socket: WebSocket, before: number) {
    socket?.send(JSON.stringify({ action: "get-hands", before: before }));
}

// listTables asks for the public tables, answered with tables; works before joining
export function listTables(socket: WebSocket) {
    socket.send(JSON.stringify({ action: "list-tables" }));
}
