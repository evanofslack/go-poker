export function sendMessage(socket: WebSocket, username: string, message: string) {
    socket.send(
        JSON.stringify({
            action: "send-message",
            params: {
                username: username,
                message: message,
            },
        })
    );
}

export function takeSeat(socket: WebSocket, username: string, position: number, buyIn: number) {
    socket.send(
        JSON.stringify({
            action: "take-seat",
            params: {
                username: username,
                position: position,
                buyIn: buyIn,
            },
        })
    );
}

export function startGame(socket: WebSocket) {
    socket.send(
        JSON.stringify({
            action: "start-game",
            params: {},
        })
    );
}

export function dealGame(socket: WebSocket) {
    socket.send(
        JSON.stringify({
            action: "deal-game",
            params: {},
        })
    );
}

export function newPlayer(socket: WebSocket, username: string) {
    socket?.send(
        JSON.stringify({
            action: "new-player",
            params: { username: username },
        })
    );
}

export function playerCall(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-call",
            params: {},
        })
    );
}

export function playerCheck(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-check",
            params: {},
        })
    );
}

export function playerRaise(socket: WebSocket, amount: number) {
    socket?.send(
        JSON.stringify({
            action: "player-raise",
            params: { amount: amount },
        })
    );
}

export function playerFold(socket: WebSocket) {
    socket?.send(
        JSON.stringify({
            action: "player-fold",
            params: {},
        })
    );
}
