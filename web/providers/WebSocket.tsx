import { createContext, ReactChild, useEffect, useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { Message, Game, Log } from "../interfaces";

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
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const { appState, dispatch } = useContext(AppContext);

    useEffect(() => {
        // WebSocket api is browser side only.
        const isBrowser = typeof window !== "undefined";

        let wsUrl = '';
        if (isBrowser) {
            wsUrl = `${window.location.origin.replace('http', 'ws')}/ws`;
            console.log("websocket url: ", wsUrl);
        }

        const _socket = isBrowser ? new WebSocket(wsUrl) : null;

        if (_socket) {
            _socket.onopen = () => {
                console.log("websocket connected");
            };
            _socket.onclose = () => {
                console.log("websocket disconnected");
            };
            _socket.onerror = (error) => {
                console.error("websocket error: ", error);
            };
        }
        setSocket(_socket);

        return () => {
            socket?.close();
        };
    }, [dispatch]);

    if (socket) {
        socket.onmessage = (e) => {
            let event = JSON.parse(e.data);
            switch (event.action) {
                case "new-message":
                    let newMessage: Message = {
                        name: event.username,
                        message: event.message,
                        timestamp: event.timestamp,
                    };
                    dispatch({ type: "addMessage", payload: newMessage });
                    return;
                case "new-log":
                    let newLog: Log = {
                        message: event.message,
                        timestamp: event.timestamp,
                    };
                    dispatch({ type: "addLog", payload: newLog });
                    return;
                case "update-game":
                    let newGame: Game = {
                        running: event.game.running,
                        dealer: event.game.dealer,
                        action: event.game.action,
                        utg: event.game.utg,
                        sb: event.game.sb,
                        bb: event.game.bb,
                        communityCards: event.game.communityCards,
                        stage: event.game.stage,
                        betting: event.game.betting,
                        config: event.game.config,
                        players: event.game.players,
                        pots: event.game.pots,
                        minRaise: event.game.minRaise,
                        readyCount: event.game.readyCount,
                    };
                    dispatch({ type: "updateGame", payload: newGame });
                    return;
                case "update-player-uuid":
                    dispatch({ type: "updatePlayerID", payload: event.uuid });
                    return;
                default:
                    throw new Error();
            }
        };
    }

    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
