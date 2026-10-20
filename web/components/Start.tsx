import { useContext } from "react";
import { startGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { Player } from "../interfaces/index";
import Button from "./Button";

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
            <Button className="m-10 text-2xl" title="need 2 or more players to start" disabled>
                start
            </Button>
        );
    }

    if (!game.running && readyPlayers.length >= 2) {
        return (
            <Button className="m-10 text-2xl" onClick={() => handleStartGame(socket)}>
                start
            </Button>
        );
    }

    return null;
}
