import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { playerCall } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";

export default function Input() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const handleCall = () => {
        if (socket) {
            playerCall(socket);
        }
    };

    if (!appState.game || appState.game.players.length === 0) return null;

    const action = appState.clientID === appState.game.players[appState.game.action].id;

    if (action) {
        return (
            <div className="bg-gray-600 p-4 text-white">
                <div className="flex flex-col bg-gray-800 p-4">
                    <button onClick={handleCall}>Call</button>
                    <button>Check</button>
                    <button>Raise</button>
                    <button>Fold</button>
                </div>
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
