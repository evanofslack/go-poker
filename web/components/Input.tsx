import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { bettingInfo } from "../lib/betting";
import Button from "./Button";
import RaiseInput from "./RaiseInput";

const btn = "flex-1 px-2 sm:flex-none sm:px-4 sm:text-xl";

export default function Input() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);

    const game = appState.game;
    if (!socket || !game || !game.betting) return null;

    if (appState.connection !== "open") {
        return <p className="py-2 text-subtle sm:text-xl">reconnecting…</p>;
    }

    const b = bettingInfo(game);
    if (b.player.uuid !== appState.clientID) {
        return <p className="py-2 text-subtle sm:text-xl">waiting for {b.player.username}</p>;
    }

    // the server takes the chips added, not the new total
    const raiseTo = (to: number) => {
        playerRaise(socket, to - b.player.bet);
        setShowRaise(false);
    };

    if (showRaise) {
        return <RaiseInput b={b} onRaise={raiseTo} onClose={() => setShowRaise(false)} />;
    }

    let callLabel = "check";
    if (b.toCall > 0) {
        callLabel = b.toCall === b.player.stack ? `all in ${b.toCall}` : `call ${b.toCall}`;
    }

    return (
        <div className="flex w-full gap-2 sm:w-auto">
            <Button variant="danger" className={btn} onClick={() => playerFold(socket)}>
                fold
            </Button>
            <Button
                variant="primary"
                className={btn}
                onClick={() => (b.toCall > 0 ? playerCall(socket) : playerCheck(socket))}
            >
                {callLabel}
            </Button>
            {b.canRaise && b.fullRaise && (
                <Button variant="primary" className={btn} onClick={() => setShowRaise(true)}>
                    {b.opening ? "bet" : "raise"}
                </Button>
            )}
            {b.canRaise && !b.fullRaise && (
                <Button variant="primary" className={btn} onClick={() => raiseTo(b.maxTo)}>
                    all in {b.maxTo}
                </Button>
            )}
        </div>
    );
}
