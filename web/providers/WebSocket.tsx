import { createContext, ReactNode, useEffect, useState, useContext, useRef } from "react";
import { AppContext } from "../providers/AppStore";
import { GameEvent } from "../interfaces";
import type { ChatOut, ErrorOut, HandHistory, Joined, TableUpdate, Tables, TokenOut, Welcome } from "../interfaces/server";
import { joinTable, leaveTable } from "../actions/actions";
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
            const msg = JSON.parse(e.data);
            switch (msg.action as string) {
                case "chat": {
                    const m = msg as ChatOut;
                    dispatch({ type: "addMessage", payload: { name: m.name, message: m.message, time: m.time } });
                    return;
                }
                case "tables":
                    dispatch({ type: "setTables", payload: (msg as Tables).tables });
                    return;
                case "hand-history": {
                    const m = msg as HandHistory;
                    dispatch({ type: "addHistory", payload: { hands: m.hands, more: m.more } });
                    return;
                }
                case "welcome": {
                    const m = msg as Welcome;
                    dispatch({
                        type: "welcome",
                        payload: {
                            view: m.view,
                            logs: m.logs,
                            hands: m.stats,
                            totals: m.totals,
                            history: m.history,
                            more: m.more,
                        },
                    });
                    return;
                }
                case "table-update": {
                    const m = msg as TableUpdate;
                    dispatch({ type: "updateView", payload: m.view });
                    for (const ev of m.events as GameEvent[]) {
                        if (ev.type === "log") {
                            dispatch({ type: "addLog", payload: ev.data.log });
                        } else if (ev.type === "hand-done") {
                            dispatch({ type: "handDone", payload: ev.data });
                        } else if (ev.type === "paid") {
                            const p = ev.data.payments;
                            const total = p.reduce((sum, x) => sum + x.amount, 0);
                            const message =
                                p[0].kind === "stand-up"
                                    ? `${p[0].from} pays the stand-up: ${total} to ${p.length === 1 ? "1 player" : `${p.length} players`}`
                                    : `${p[0].to} wins the ${p[0].kind}: ${total}`;
                            dispatch({ type: "addToast", payload: { message, kind: "info" } });
                        } else if (ev.type === "hand-updated") {
                            dispatch({ type: "handUpdated", payload: ev.data.record });
                        }
                    }
                    return;
                }
                case "error": {
                    const m = msg as ErrorOut;
                    if (m.code === "restarting") {
                        // the table is saved: keep the session and rejoin once reconnected
                        dispatch({ type: "addToast", payload: { message: m.message, kind: "info" } });
                        return;
                    }
                    // rejected actions only go to this client, show them in the log
                    dispatch({ type: "addLog", payload: { message: "error: " + m.message, time: Date.now() } });
                    dispatch({ type: "setError", payload: m.message });
                    if (m.code === "removed" || m.code === "closed") {
                        // the host removed us, or the table closed: leave, and don't rejoin on reconnect
                        if (appState.table) leaveTable(socket, appState.table);
                        saveSession(null);
                        window.history.replaceState(null, "", window.location.pathname);
                        dispatch({ type: "leftTable" });
                        // the join form shows why
                        dispatch({ type: "setError", payload: m.message });
                        return;
                    }
                    if (m.code === "password-required" || m.code === "wrong-password") {
                        dispatch({ type: "setNeedPassword", payload: true });
                    }
                    // the join form shows its own errors
                    if (appState.table) {
                        dispatch({ type: "addToast", payload: { message: m.message, kind: "error" } });
                    }
                    if (rejoining.current) {
                        rejoining.current = false;
                        saveSession(null);
                        dispatch({ type: "resetGame" });
                    }
                    return;
                }
                case "joined-table": {
                    const m = msg as Joined;
                    rejoining.current = false;
                    // keep the token when rejoining the same seat
                    const prev = loadSession();
                    const same = prev?.table === m.tablename && prev?.username === m.username;
                    saveSession({
                        table: m.tablename,
                        username: m.username,
                        token: same ? prev?.token : undefined,
                        password: m.password,
                    });
                    dispatch({ type: "setError", payload: null });
                    dispatch({ type: "setNeedPassword", payload: false });
                    dispatch({ type: "setPassword", payload: m.password ?? null });
                    dispatch({ type: "setTablename", payload: m.tablename });
                    // the address bar doubles as the invite link
                    window.history.replaceState(null, "", inviteLink(m.tablename));
                    dispatch({ type: "setUsername", payload: m.username });
                    return;
                }
                case "token": {
                    // an empty token means the seat was given up
                    const session = loadSession();
                    if (session) {
                        saveSession({ ...session, token: (msg as TokenOut).token || undefined });
                    }
                    return;
                }
                default:
                    console.warn("unknown message", msg.action);
            }
        };
    }

    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
