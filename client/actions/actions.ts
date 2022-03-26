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
