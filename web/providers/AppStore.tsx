import { createContext, useReducer, ReactNode } from "react";
import { AppState, Message, Game, Log, Connection, Toast } from "../interfaces";

const initialState: AppState = {
    messages: [],
    logs: [],
    username: null,
    clientID: null,
    table: null,
    game: null,
    error: null,
    connection: "connecting",
    toasts: [],
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
    | { type: "setError"; payload: string | null }
    | { type: "setConnection"; payload: Connection }
    | { type: "addToast"; payload: Omit<Toast, "id"> }
    | { type: "removeToast"; payload: number };

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
        case "resetGame":
            return { ...state, clientID: null, username: null, game: null };
        case "leftTable":
            return {
                ...initialState,
                connection: state.connection,
                toasts: state.toasts,
            };
        case "updatePlayerID":
            return { ...state, clientID: action.payload };
        case "setTablename":
            return { ...state, table: action.payload };
        case "setError":
            return { ...state, error: action.payload };
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
