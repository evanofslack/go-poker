import { createContext, useReducer, ReactNode } from "react";
import {
    AppState,
    Message,
    Log,
    Connection,
    Toast,
    HandStats,
    HandRecord,
    PlayerTotals,
    TableSummary,
    View,
} from "../interfaces";

// maxHands matches the server's cap
const maxHands = 1000;

const initialState: AppState = {
    messages: [],
    logs: [],
    username: null,
    table: null,
    password: null,
    view: null,
    error: null,
    needPassword: false,
    connection: "connecting",
    toasts: [],
    hands: [],
    totals: {},
    history: [],
    moreHistory: false,
    tables: [],
};

let nextToastID = 0;

type ACTIONTYPE =
    | { type: "addMessage"; payload: Message }
    | { type: "addLog"; payload: Log }
    | { type: "setUsername"; payload: string }
    | { type: "updateView"; payload: View }
    | {
          type: "welcome";
          payload: {
              view: View;
              logs: Log[];
              hands: HandStats[];
              totals: Record<string, PlayerTotals>;
              history: HandRecord[];
              more: boolean;
          };
      }
    | { type: "handDone"; payload: { record: HandRecord; stats: HandStats; totals: Record<string, PlayerTotals> } }
    | { type: "resetGame" }
    | { type: "leftTable" }
    | { type: "setTablename"; payload: string }
    | { type: "setPassword"; payload: string | null }
    | { type: "setNeedPassword"; payload: boolean }
    | { type: "setError"; payload: string | null }
    | { type: "setConnection"; payload: Connection }
    | { type: "addToast"; payload: Omit<Toast, "id"> }
    | { type: "removeToast"; payload: number }
    | { type: "addHistory"; payload: { hands: HandRecord[]; more: boolean } }
    | { type: "setTables"; payload: TableSummary[] };

function reducer(state: AppState, action: ACTIONTYPE): AppState {
    switch (action.type) {
        case "addMessage":
            return { ...state, messages: [...state.messages, action.payload] };
        case "addLog":
            return { ...state, logs: [...state.logs, action.payload] };
        case "setUsername":
            return { ...state, username: action.payload };
        case "updateView":
            return { ...state, view: action.payload };
        case "welcome": {
            // everything so far, replacing what a previous join left
            const { view, logs, hands, totals, history, more } = action.payload;
            return { ...state, view, logs, hands, totals, history, moreHistory: more };
        }
        case "handDone": {
            const { record, stats, totals } = action.payload;
            return {
                ...state,
                hands: [...state.hands, stats].slice(-maxHands),
                totals,
                history: [...state.history.filter((h) => h.hand !== record.hand), record],
            };
        }
        case "addHistory": {
            // older hands, answering getHands
            const { hands, more } = action.payload;
            const byHand = new Map(state.history.map((h) => [h.hand, h]));
            for (const h of hands) byHand.set(h.hand, h);
            const history = Array.from(byHand.values()).sort((a, b) => a.hand - b.hand);
            return { ...state, history, moreHistory: more };
        }
        case "resetGame":
            return { ...state, username: null, view: null };
        case "leftTable":
            return {
                ...initialState,
                connection: state.connection,
                toasts: state.toasts,
                tables: state.tables,
            };
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
