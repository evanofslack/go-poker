import { useContext } from "react";
import { resetGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Button from "./Button";

export default function Reset() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);

    // only the host can reset the table
    if (!socket || appState.game?.host !== appState.clientID) {
        return null;
    }
    return (
        <Button variant="ghost" className="px-1 text-sm" onClick={() => resetGame(socket)}>
            reset
        </Button>
    );
}
