import { SocketContext } from "../providers/WebSocket";
import { useContext } from "react";

export function useSocket() {
    const socket = useContext(SocketContext);
    return socket;
}
