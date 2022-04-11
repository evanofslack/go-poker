import { useContext, Dispatch } from "react";
import { resetGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";

function handleResetGame(socket: WebSocket | null, dispatch: Dispatch<any>) {
    if (socket) {
        resetGame(socket);
        dispatch({ type: "resetGame" });
    }
}
export default function Reset() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    return (
        <button
            className="p-4 text-zinc-600 hover:underline"
            onClick={() => handleResetGame(socket, dispatch)}
        >
            Reset
        </button>
    );
}
