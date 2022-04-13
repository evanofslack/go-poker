import { useContext } from "react";
import { startGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { Player } from "../interfaces/index";

type startProps = {
    players: (Player | null)[];
};

function handleStartGame(socket: WebSocket | null) {
    if (socket) {
        startGame(socket);
    }
}

export default function Start({ players }: startProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;
    const readyPlayers = players.filter((player) => player != null);

    if (!game) {
        return null;
    }

    if (!game.running && readyPlayers.length < 2) {
        return (
            <div
                className=" m-10 rounded-lg border border-2 border-neutral-200 p-2 px-4 py-3 text-2xl font-medium text-neutral-200 opacity-20"
                title="Must have 2 or more players to start game"
            >
                Start
            </div>
        );
    }

    if (!game.running && readyPlayers.length >= 2) {
        return (
            <button
                className=" m-10 rounded-lg border border-2 border-neutral-200 p-2 px-4 py-3 text-2xl font-medium text-neutral-200"
                onClick={() => handleStartGame(socket)}
            >
                Start
            </button>
        );
    }

    return null;
}