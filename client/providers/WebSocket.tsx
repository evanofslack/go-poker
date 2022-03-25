import { createContext, ReactChild } from "react";

export const isBrowser = typeof window !== "undefined";
const socket = isBrowser ? new WebSocket("ws://127.0.0.1:8080/ws") : null;

export const SocketContext = createContext(socket);

type SocketProviderProps = {
    children: ReactChild;
};

export function SocketProvider(props: SocketProviderProps) {
    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
