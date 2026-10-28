import { createContext, useEffect, useReducer, ReactNode } from "react";
import {
    AppState,
    Message,
    Log,
    Connection,
    HandStats,
    HandRecord,
    PlayerTotals,
    TableSummary,
    View,
} from "../interfaces";
import { MaxHands } from "../interfaces/table";
import { Prefs, loadPrefs, savePrefs, showLooks } from "../lib/prefs";

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
    hands: [],
    totals: {},
    history: [],
    moreHistory: false,
    tables: [],
    prefs: loadPrefs(),
    welcomed: 0,
};


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
    | { type: "handUpdated"; payload: HandRecord }
    | { type: "resetGame" }
    | { type: "leftTable" }
    | { type: "setTablename"; payload: string }
    | { type: "setPassword"; payload: string | null }
    | { type: "setNeedPassword"; payload: boolean }
    | { type: "setError"; payload: string | null }
    | { type: "setConnection"; payload: Connection }
    | { type: "addHistory"; payload: { hands: HandRecord[]; more: boolean } }
    | { type: "setTables"; payload: TableSummary[] }
    | { type: "setPref"; payload: Partial<Prefs> };

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
            return { ...state, view, logs, hands, totals, history, moreHistory: more, welcomed: state.welcomed + 1 };
        }
        case "handDone": {
            const { record, stats, totals } = action.payload;
            return {
                ...state,
                hands: [...state.hands, stats].slice(-MaxHands),
                totals,
                history: [...state.history.filter((h) => h.hand !== record.hand), record],
            };
        }
        case "handUpdated": {
            // someone showed their cards after the hand
            const record = action.payload;
            return { ...state, history: state.history.map((h) => (h.hand === record.hand ? record : h)) };
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
                tables: state.tables,
                prefs: state.prefs,
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
        case "setPref": {
            const prefs = { ...state.prefs, ...action.payload };
            savePrefs(prefs);
            return { ...state, prefs };
        }
        case "setConnection":
            return { ...state, connection: action.payload };
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
    useEffect(() => showLooks(appState.prefs), [appState.prefs]);

    return (
        <AppContext.Provider value={{ appState, dispatch }}>{props.children}</AppContext.Provider>
    );
}
