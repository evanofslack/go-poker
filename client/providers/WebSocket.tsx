import { createContext, ReactChild, useEffect, useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { Event, Message, Game } from "../interfaces";

/*  
WebSocket context creates a single connection to the server per client. 
It handles opening, closing, and error handling of the websocket. It also
dispatches websocket messages to update the central state store. 
*/

export const SocketContext = createContext<WebSocket | null>(null);

type SocketProviderProps = {
    children: ReactChild;
};

export function SocketProvider(props: SocketProviderProps) {
    const WS_URL = "ws://127.0.0.1:8080/ws";
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const { appState, dispatch } = useContext(AppContext);

    useEffect(() => {
        // WebSocket api is browser side only.
        const isBrowser = typeof window !== "undefined";
        const _socket = isBrowser ? new WebSocket(WS_URL) : null;

        if (_socket) {
            _socket.onopen = () => {
                console.log("websocket connected");
            };
            _socket.onclose = () => {
                console.log("websocket disconnected");
            };
        }
        setSocket(_socket);

        return () => {
            socket?.close();
        };
    }, [dispatch]);

    if (socket) {
        socket.onmessage = (e) => {
            let event: Event = JSON.parse(e.data);
            switch (event.action) {
                case "new-message":
                    let newMessage: Message = {
                        name: event.params.username,
                        message: event.params.message,
                        timestamp: event.params.timestamp,
                    };
                    dispatch({ type: "addMessage", payload: newMessage });
                    return;
                case "update-game":
                    let newGame: Game = {
                        dealer: event.params.dealer,
                        action: event.params.action,
                        utg: event.params.utg,
                        sb: event.params.sb,
                        bb: event.params.bb,
                        communityCards: event.params.communityCards,
                        stage: event.params.stage,
                        betting: event.params.betting,
                        config: event.params.config,
                        players: event.params.players,
                        pots: event.params.pots,
                        minRaise: event.params.minRaise,
                        readyCount: event.params.readyCount,
                    };
                    dispatch({ type: "updateGame", payload: newGame });
                    return;
                case "update-player-id":
                    dispatch({ type: "updatePlayerID", payload: event.params.id });
                    return;
                default:
                    throw new Error();
            }
        };
    }

    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
