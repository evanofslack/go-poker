import { createContext, useReducer, ReactChild } from "react";
import { AppState, Message, Game, Seat } from "../interfaces";

const initialState: AppState = {
    messages: [],
    username: null,
    seatID: null,
    game: null,
};

type ACTIONTYPE =
    | { type: "addMessage"; payload: Message }
    | { type: "setUsername"; payload: string }
    | { type: "updateGame"; payload: Game };

function reducer(state: AppState, action: ACTIONTYPE) {
    switch (action.type) {
        case "addMessage":
            return { ...state, messages: [...state.messages, action.payload] };
        case "setUsername":
            return { ...state, username: action.payload };
        case "updateGame":
            return { ...state, game: action.payload };
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
