import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";

export default function Input() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const handleCall = () => {
        if (socket) {
            playerCall(socket);
        }
    };
    const handleRaise = (amount: number) => {
        if (socket) {
            playerRaise(socket, amount);
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
        return (
            <div className="flex flex-row p-6">
                <InputButton
                    action={handleCall}
                    title={canCall ? "call" : "call (" + callAmount + ")"}
                    disabled={canCall}
                />
                <InputButton action={() => handleRaise(1000)} title={"raise"} disabled={false} />
                <InputButton action={handleCheck} title={"check"} disabled={!canCheck} />
                <InputButton action={handleFold} title={"fold"} disabled={false} />
            </div>
        );
    }

    return (
        <div className="bg-gray-600 p-4 text-white">
            <div className="flex flex-col bg-gray-800 p-4">
                <p>Not your turn bitch</p>
            </div>
        </div>
    );
}
