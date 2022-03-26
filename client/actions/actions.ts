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

export function takeSeat(socket: WebSocket, username: string, buyIn: number) {
    socket.send(
        JSON.stringify({
            action: "take-seat",
            params: {
                username: username,
                buyIn: buyIn,
            },
        })
    );
}
