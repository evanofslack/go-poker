import { useContext } from "react";
import { startGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Button from "./Button";

function handleStartGame(socket: WebSocket | null) {
    if (socket) {
        startGame(socket);
    }
}

export default function Start() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;

    if (!game || game.ended) {
        return null;
    }
    if (!game.running && game.paused) {
        return (
            <Button className="text-lg" title="the host paused the game" disabled>
                start
            </Button>
        );
    }

    if (!game.running && game.players.length < 2) {
        return (
            <Button className="text-lg" title="need 2 or more players to start" disabled>
                start
            </Button>
        );
    }

    if (!game.running && game.players.length >= 2) {
        return (
            <Button className="text-lg" onClick={() => handleStartGame(socket)}>
                start
            </Button>
        );
    }

    return null;
}
