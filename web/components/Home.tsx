import { useContext, useState } from "react";
import { leaveTable } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { clearSession } from "../providers/WebSocket";
import Confirm from "./Confirm";

// Home leaves the table and goes back to the join form, asking first if you have a seat
export default function Home() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [confirming, setConfirming] = useState(false);
    const game = appState.view;
    const player = game?.players.find((p) => p.seat === game.you);

    const leave = () => {
        if (socket && appState.table) leaveTable(socket, appState.table);
        clearSession();
        window.history.replaceState(null, "", window.location.pathname);
        dispatch({ type: "leftTable" });
    };

    let message = "you'll give up your seat.";
    if (game?.betting && player?.in) message = "you'll fold this hand and give up your seat.";

    return (
        <>
            <button
                className="px-1 py-1 font-semibold text-fg hover:text-accent sm:text-lg"
                onClick={() => (player ? setConfirming(true) : leave())}
            >
                GoPoker
            </button>
            {confirming && (
                <Confirm
                    title="leave the table?"
                    message={message}
                    action="leave"
                    onConfirm={leave}
                    onCancel={() => setConfirming(false)}
                />
            )}
        </>
    );
}
