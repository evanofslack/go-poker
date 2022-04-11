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
    console.log(seatID);
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

export function dealGame(socket: WebSocket) {
    socket.send(
        JSON.stringify({
            action: "deal-game",
        })
    );
}

export function newPlayer(socket: WebSocket, username: string) {
    socket?.send(
        JSON.stringify({
            action: "new-player",
            username: username,
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
