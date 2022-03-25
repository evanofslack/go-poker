import { createContext, useReducer, ReactChild } from "react";
import { AppState, Message } from "../interfaces";

const initialState: AppState = {
    messages: [],
    username: "",
}

type ACTIONTYPE =
  | { type: "addMessage"; payload: Message }
  | { type: "setUsername"; payload: string }

function reducer(state: AppState, action: ACTIONTYPE) {
    switch (action.type) {
      case 'addMessage':
        return {...state, messages: [...state.messages, action.payload ]};
      case 'setUsername':
        return {...state, username:  action.payload };
      default:
        throw new Error();
    }
  }

export const AppContext = createContext<{
    state: AppState;
    dispatch: React.Dispatch<any>;
  }>({state: initialState, dispatch: () => null})

type StoreProviderProps = {
    children: ReactChild
}

export function AppStateProvider(props: StoreProviderProps) {

    const [state, dispatch] = useReducer(reducer, initialState);

    return <AppContext.Provider value={{state, dispatch}}>{props.children}</AppContext.Provider>
}