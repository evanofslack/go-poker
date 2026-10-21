import { createContext, ReactNode, useEffect, useState, useContext, useRef } from "react";
import { AppContext } from "../providers/AppStore";
import { Message, Game, Log } from "../interfaces";
import { joinTable } from "../actions/actions";
import { inviteLink } from "../lib/invite";

// Session lets a tab rejoin its table, and reclaim its seat with the token,
// after a dropped connection or a reload. The password lets a spectator back in.
type Session = { table: string; username: string; token?: string; password?: string };
const sessionKey = "session";

function loadSession(): Session | null {
    try {
        return JSON.parse(sessionStorage.getItem(sessionKey) || "null");
    } catch {
        return null;
    }
}

// clearSession forgets the table, so a reload goes back to the join form
export function clearSession() {
    saveSession(null);
}

function saveSession(session: Session | null) {
    try {
        if (session) {
            sessionStorage.setItem(sessionKey, JSON.stringify(session));
        } else {
            sessionStorage.removeItem(sessionKey);
        }
    } catch {}
}

/*  
WebSocket context creates a single connection to the server per client. 
It handles opening, closing, and error handling of the websocket. It also
dispatches websocket messages to update the central state store. 
*/

export const SocketContext = createContext<WebSocket | null>(null);

type SocketProviderProps = {
    children: ReactNode;
};

export function SocketProvider(props: SocketProviderProps) {
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const { appState, dispatch } = useContext(AppContext);

    // true while rejoining after a reconnect, so a failed rejoin can go back to the join form
    const rejoining = useRef(false);

    useEffect(() => {
        // WebSocket api is browser side only.
        if (typeof window === "undefined") {
            return;
        }
        // NEXT_PUBLIC_WS_URL points next dev at the go server, otherwise same origin
        const wsUrl =
            process.env.NEXT_PUBLIC_WS_URL || `${window.location.origin.replace("http", "ws")}/ws`;

        let ws: WebSocket;
        let retry: ReturnType<typeof setTimeout>;
        let delay = 1000;
        let stopped = false;
        let wasOpen = false;

        // connect, and reconnect with backoff whenever the connection drops
        const connect = () => {
            ws = new WebSocket(wsUrl);
            ws.onopen = () => {
                delay = 1000;
                dispatch({ type: "setConnection", payload: "open" });
                if (wasOpen) {
                    dispatch({ type: "addToast", payload: { message: "reconnected", kind: "info" } });
                }
                wasOpen = true;
                const session = loadSession();
                if (session) {
                    rejoining.current = true;
                    joinTable(ws, session.table, session.username, session.token, session.password);
                }
            };
            ws.onclose = () => {
                if (stopped) {
                    return;
                }
                setSocket(null);
                if (wasOpen) {
                    dispatch({ type: "setConnection", payload: "reconnecting" });
                }
                retry = setTimeout(connect, delay);
                delay = Math.min(delay * 2, 10000);
            };
            ws.onerror = (error) => {
                console.error("websocket error: ", error);
            };
            setSocket(ws);
        };
        connect();

        return () => {
            stopped = true;
            clearTimeout(retry);
            ws.close();
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
                        host: event.host,
                        turnEndsIn: event.turnEndsIn,
                        sitOutNext: event.sitOutNext,
                        settings: event.settings,
                        hasPassword: event.hasPassword,
                        watching: event.watching,
                    };
                    dispatch({ type: "updateGame", payload: newGame });
                    return;
                case "error":
                    // rejected actions only go to this client, show them in the log
                    let errorLog: Log = {
                        message: "error: " + event.message,
                        timestamp: event.timestamp,
                    };
                    dispatch({ type: "addLog", payload: errorLog });
                    dispatch({ type: "setError", payload: event.message });
                    if (event.code === "password-required" || event.code === "wrong-password") {
                        dispatch({ type: "setNeedPassword", payload: true });
                    }
                    // the join form shows its own errors
                    if (appState.table) {
                        dispatch({ type: "addToast", payload: { message: event.message, kind: "error" } });
                    }
                    if (rejoining.current) {
                        rejoining.current = false;
                        saveSession(null);
                        dispatch({ type: "resetGame" });
                    }
                    return;
                case "joined-table": {
                    rejoining.current = false;
                    // keep the token when rejoining the same seat
                    const prev = loadSession();
                    const same = prev?.table === event.tablename && prev?.username === event.username;
                    saveSession({
                        table: event.tablename,
                        username: event.username,
                        token: same ? prev?.token : undefined,
                        password: event.password,
                    });
                    dispatch({ type: "setError", payload: null });
                    dispatch({ type: "setNeedPassword", payload: false });
                    dispatch({ type: "setPassword", payload: event.password ?? null });
                    dispatch({ type: "setTablename", payload: event.tablename });
                    // the address bar doubles as the invite link
                    window.history.replaceState(null, "", inviteLink(event.tablename));
                    dispatch({ type: "updatePlayerID", payload: event.uuid });
                    dispatch({ type: "setUsername", payload: event.username });
                    return;
                }
                case "update-player-uuid": {
                    // an empty token means the seat was given up
                    const session = loadSession();
                    if (session) {
                        saveSession({ ...session, token: event.token || undefined });
                    }
                    dispatch({ type: "updatePlayerID", payload: event.uuid });
                    return;
                }
                default:
                    throw new Error();
            }
        };
    }

    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
