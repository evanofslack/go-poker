import { createContext, useReducer, ReactNode } from "react";
import { AppState, Message, Game, Log, Connection, Toast, HandStats, HandRecord, TableSummary } from "../interfaces";

// maxHands matches the server's cap
const maxHands = 1000;

const initialState: AppState = {
    messages: [],
    logs: [],
    username: null,
    clientID: null,
    table: null,
    password: null,
    game: null,
    error: null,
    needPassword: false,
    connection: "connecting",
    toasts: [],
    hands: [],
    history: [],
    moreHistory: false,
    tables: [],
};

let nextToastID = 0;

type ACTIONTYPE =
    | { type: "addMessage"; payload: Message }
    | { type: "addLog"; payload: Log }
    | { type: "setUsername"; payload: string }
    | { type: "updateGame"; payload: Game }
    | { type: "resetGame" }
    | { type: "leftTable" }
    | { type: "updatePlayerID"; payload: string }
    | { type: "setTablename"; payload: string }
    | { type: "setPassword"; payload: string | null }
    | { type: "setNeedPassword"; payload: boolean }
    | { type: "setError"; payload: string | null }
    | { type: "setConnection"; payload: Connection }
    | { type: "addToast"; payload: Omit<Toast, "id"> }
    | { type: "removeToast"; payload: number }
    | { type: "addHands"; payload: { hands: HandStats[]; full: boolean } }
    | { type: "addHistory"; payload: { hands: HandRecord[]; full: boolean; more: boolean } }
    | { type: "setTables"; payload: TableSummary[] };

function reducer(state: AppState, action: ACTIONTYPE) {
    switch (action.type) {
        case "addMessage":
            return { ...state, messages: [...state.messages, action.payload] };
        case "addLog":
            return { ...state, logs: [...state.logs, action.payload] };
        case "setUsername":
            return { ...state, username: action.payload };
        case "updateGame":
            return { ...state, game: action.payload };
        case "addHands": {
            const { hands, full } = action.payload;
            return { ...state, hands: (full ? hands : [...state.hands, ...hands]).slice(-maxHands) };
        }
        case "addHistory": {
            const { hands, full, more } = action.payload;
            const byHand = new Map((full ? [] : state.history).map((h) => [h.hand, h]));
            for (const h of hands) byHand.set(h.hand, h);
            const history = Array.from(byHand.values()).sort((a, b) => a.hand - b.hand);
            // more is about hands older than this message's first; an empty
            // answer to getHands means there are none
            const oldest = full || !hands.length || !state.history.length || hands[0].hand <= state.history[0].hand;
            return { ...state, history, moreHistory: oldest ? more : state.moreHistory };
        }
        case "resetGame":
            return { ...state, clientID: null, username: null, game: null };
        case "leftTable":
            return {
                ...initialState,
                connection: state.connection,
                toasts: state.toasts,
                tables: state.tables,
            };
        case "updatePlayerID":
            return { ...state, clientID: action.payload };
        case "setTablename":
            return { ...state, table: action.payload };
        case "setPassword":
            return { ...state, password: action.payload };
        case "setNeedPassword":
            return { ...state, needPassword: action.payload };
        case "setError":
            return { ...state, error: action.payload };
        case "setTables":
            return { ...state, tables: action.payload };
        case "setConnection":
            return { ...state, connection: action.payload };
        case "addToast": {
            // keep the last few so a burst of errors doesn't fill the screen
            const toast = { ...action.payload, id: nextToastID++ };
            return { ...state, toasts: [...state.toasts.slice(-2), toast] };
        }
        case "removeToast":
            return { ...state, toasts: state.toasts.filter((t) => t.id !== action.payload) };
        default:
            throw new Error();
    }
}

export const AppContext = createContext<{
    appState: AppState;
    dispatch: React.Dispatch<any>;
}>({ appState: initialState, dispatch: () => null });

type StoreProviderProps = {
    children: ReactNode;
};

export function AppStoreProvider(props: StoreProviderProps) {
    const [appState, dispatch] = useReducer(reducer, initialState);

    return (
        <AppContext.Provider value={{ appState, dispatch }}>{props.children}</AppContext.Provider>
    );
}
