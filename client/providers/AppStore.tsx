import { createContext, useReducer, ReactChild } from "react";
import { AppState, Message, Game } from "../interfaces";

const initialState: AppState = {
    messages: [],
    username: null,
    clientID: null,
    game: null,
};

type ACTIONTYPE =
    | { type: "addMessage"; payload: Message }
    | { type: "setUsername"; payload: string }
    | { type: "updateGame"; payload: Game }
    | { type: "resetGame" }
    | { type: "updatePlayerID"; payload: string };

function reducer(state: AppState, action: ACTIONTYPE) {
    switch (action.type) {
        case "addMessage":
            return { ...state, messages: [...state.messages, action.payload] };
        case "setUsername":
            return { ...state, username: action.payload };
        case "updateGame":
            return { ...state, game: action.payload };
        case "resetGame":
            return { ...state, clientID: null, username: null, game: null };
        case "updatePlayerID":
            return { ...state, clientID: action.payload };
        default:
            throw new Error();
    }
}

export const AppContext = createContext<{
    appState: AppState;
    dispatch: React.Dispatch<any>;
}>({ appState: initialState, dispatch: () => null });

type StoreProviderProps = {
    children: ReactChild;
};

export function AppStoreProvider(props: StoreProviderProps) {
    const [appState, dispatch] = useReducer(reducer, initialState);

    return (
        <AppContext.Provider value={{ appState, dispatch }}>{props.children}</AppContext.Provider>
    );
}
