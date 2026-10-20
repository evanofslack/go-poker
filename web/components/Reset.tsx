import { useContext } from "react";
import { resetGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";

export default function Reset() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);

    // only the host can reset the table
    if (!socket || appState.game?.host !== appState.clientID) {
        return null;
    }
    return (
        <button
            className="m-2 p-2 text-zinc-800 hover:text-zinc-700"
            onClick={() => resetGame(socket)}
        >
            reset
        </button>
    );
}
