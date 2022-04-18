import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, sendLog } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";
import RaiseInput from "./RaiseInput";

export default function Input() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);

    const handleCall = (user: string | null, amount: number) => {
        if (socket) {
            let callMessage = user + " calls " + amount;
            sendLog(socket, callMessage);
            playerCall(socket);
        }
    };
    const handleCheck = (user: string | null) => {
        if (socket) {
            let checkMessage = user + " checks";
            sendLog(socket, checkMessage);
            playerCheck(socket);
        }
    };
    const handleFold = (user: string | null) => {
        if (socket) {
            let foldMessage = user + " folds";
            sendLog(socket, foldMessage);
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
                <InputButton
                    action={() => handleCall(appState.username, callAmount)}
                    title={canCall ? "call" : "call (" + callAmount + ")"}
                    disabled={canCall}
                />
                <InputButton
                    action={() => setShowRaise(!showRaise)}
                    title={"bet"}
                    disabled={false}
                />
                <InputButton
                    action={() => handleCheck(appState.username)}
                    title={"check"}
                    disabled={!canCheck}
                />
                <InputButton
                    action={() => handleFold(appState.username)}
                    title={"fold"}
                    disabled={false}
                />
            </div>
        );
    }

    return (
        <div className="flex flex-row p-6">
            <InputButton
                action={() => handleCall(appState.username, callAmount)}
                title={canCall ? "call" : "call (" + callAmount + ")"}
                disabled={true}
            />
            <InputButton action={() => setShowRaise(!showRaise)} title={"bet"} disabled={true} />
            <InputButton
                action={() => handleCheck(appState.username)}
                title={"check"}
                disabled={true}
            />
            <InputButton
                action={() => handleFold(appState.username)}
                title={"fold"}
                disabled={true}
            />
        </div>
    );
}
