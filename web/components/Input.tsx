import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { bettingInfo } from "../lib/betting";
import Button from "./Button";
import RaiseInput from "./RaiseInput";

export default function Input() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);

    const game = appState.game;
    if (!socket || !game || !game.betting) return null;

    const b = bettingInfo(game);
    if (b.player.uuid !== appState.clientID) {
        return <p className="p-6 text-xl text-subtle">waiting for {b.player.username}</p>;
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
        <div className="flex flex-row p-6">
            <Button variant="danger" className="mx-1 text-xl" onClick={() => playerFold(socket)}>
                fold
            </Button>
            <Button
                variant="primary"
                className="mx-1 text-xl"
                onClick={() => (b.toCall > 0 ? playerCall(socket) : playerCheck(socket))}
            >
                {callLabel}
            </Button>
            {b.canRaise && b.fullRaise && (
                <Button variant="primary" className="mx-1 text-xl" onClick={() => setShowRaise(true)}>
                    {b.opening ? "bet" : "raise"}
                </Button>
            )}
            {b.canRaise && !b.fullRaise && (
                <Button variant="primary" className="mx-1 text-xl" onClick={() => raiseTo(b.maxTo)}>
                    all in {b.maxTo}
                </Button>
            )}
        </div>
    );
}
