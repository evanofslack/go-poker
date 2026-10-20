import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import Button from "./Button";
import RaiseInput from "./RaiseInput";

export default function Input() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);

    const handleCall = (user: string | null, amount: number) => {
        if (socket) {
            playerCall(socket);
        }
    };
    const handleCheck = (user: string | null) => {
        if (socket) {
            playerCheck(socket);
        }
    };
    const handleFold = (user: string | null) => {
        if (socket) {
            playerFold(socket);
        }
    };

    if (!appState.game || appState.game.betting == false) return null;

    const action = appState.clientID === appState.game.players[appState.game.action].uuid;

    const player = appState.game.players[appState.game.action];
    const playerBets = appState.game.players.map((player) => player.bet);
    const maxBet = Math.max(...playerBets);

    const canCheck = player.bet >= maxBet;
    const canCall = maxBet - player.bet === 0;
    const callAmount = maxBet - player.bet < player.stack ? maxBet - player.bet : player.stack;

    if (action) {
        if (showRaise) {
            return <RaiseInput setShowRaise={setShowRaise} showRaise={showRaise} />;
        }
        return (
            <div className="flex flex-row p-6">
                <Button
                    variant="primary"
                    className="mx-1 text-xl"
                    onClick={() => handleCall(appState.username, callAmount)}
                    disabled={canCall}
                >
                    {canCall ? "call" : "call (" + callAmount + ")"}
                </Button>
                <Button
                    variant="primary"
                    className="mx-1 text-xl"
                    onClick={() => setShowRaise(!showRaise)}
                    disabled={false}
                >
                    bet
                </Button>
                <Button
                    variant="primary"
                    className="mx-1 text-xl"
                    onClick={() => handleCheck(appState.username)}
                    disabled={!canCheck}
                >
                    check
                </Button>
                <Button
                    variant="danger"
                    className="mx-1 text-xl"
                    onClick={() => handleFold(appState.username)}
                    disabled={false}
                >
                    fold
                </Button>
            </div>
        );
    }

    return (
        <div className="flex flex-row p-6">
            <Button
                variant="primary"
                className="mx-1 text-xl"
                onClick={() => handleCall(appState.username, callAmount)}
                disabled={true}
            >
                {canCall ? "call" : "call (" + callAmount + ")"}
            </Button>
            <Button
                variant="primary"
                className="mx-1 text-xl"
                onClick={() => setShowRaise(!showRaise)}
                disabled={true}
            >
                bet
            </Button>
            <Button
                variant="primary"
                className="mx-1 text-xl"
                onClick={() => handleCheck(appState.username)}
                disabled={true}
            >
                check
            </Button>
            <Button
                variant="danger"
                className="mx-1 text-xl"
                onClick={() => handleFold(appState.username)}
                disabled={true}
            >
                fold
            </Button>
        </div>
    );
}
