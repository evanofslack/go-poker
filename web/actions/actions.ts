export function createTable(socket: WebSocket, tablename: string, username: string) {
    socket.send(
        JSON.stringify({
            action: "create-table",
            tablename: tablename,
            username: username,
        })
    );
}

export function joinTable(socket: WebSocket, tablename: string, username: string, token?: string) {
    socket.send(
        JSON.stringify({
            action: "join-table",
            tablename: tablename,
            username: username,
            token: token,
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

export function resetGame(socket: WebSocket) {
    socket.send(
        JSON.stringify({
            action: "reset-game",
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
