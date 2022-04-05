import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";
import RaiseInput from "./RaiseInput";

export default function Input() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);

    const handleCall = () => {
        if (socket) {
            playerCall(socket);
        }
    };
    const handleCheck = () => {
        if (socket) {
            playerCheck(socket);
        }
    };
    const handleFold = () => {
        if (socket) {
            playerFold(socket);
        }
    };

    if (!appState.game || appState.game.betting == false) return null;

    const action = appState.clientID === appState.game.players[appState.game.action].id;

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
                    action={handleCall}
                    title={canCall ? "call" : "call (" + callAmount + ")"}
                    disabled={canCall}
                />
                <InputButton
                    action={() => setShowRaise(!showRaise)}
                    title={"bet"}
                    disabled={false}
                />
                <InputButton action={handleCheck} title={"check"} disabled={!canCheck} />
                <InputButton action={handleFold} title={"fold"} disabled={false} />
            </div>
        );
    }

    return (
        <div className="flex flex-row p-6">
            <InputButton
                action={handleCall}
                title={canCall ? "call" : "call (" + callAmount + ")"}
                disabled={true}
            />
            <InputButton action={() => setShowRaise(!showRaise)} title={"bet"} disabled={true} />
            <InputButton action={handleCheck} title={"check"} disabled={true} />
            <InputButton action={handleFold} title={"fold"} disabled={true} />
        </div>
    );
}
