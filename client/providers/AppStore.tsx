import { createContext, useReducer, ReactChild } from "react";
import { AppState, Message } from "../interfaces";

const initialState: AppState = {
    messages: [],
    username: "",
};

type ACTIONTYPE =
    | { type: "addMessage"; payload: Message }
    | { type: "setUsername"; payload: string };

function reducer(state: AppState, action: ACTIONTYPE) {
    switch (action.type) {
        case "addMessage":
            return { ...state, messages: [...state.messages, action.payload] };
        case "setUsername":
            return { ...state, username: action.payload };
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
