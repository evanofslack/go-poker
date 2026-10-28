import { useContext } from "react";
import { startGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { Button } from "@mantine/core";
import { bar } from "../lib/theme";

function handleStartGame(socket: WebSocket | null) {
    if (socket) {
        startGame(socket);
    }
}

export default function Start() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const view = appState.view;

    if (!view || view.ended) {
        return null;
    }
    if (!view.running && view.paused) {
        return (
            <Button variant="filled" size="md" className={bar} title="the host paused the game" disabled>
                start
            </Button>
        );
    }

    if (!view.running && view.players.filter((p) => !p.sittingOut).length < 2) {
        return (
            <Button variant="filled" size="md" className={bar} title="need 2 or more players to start" disabled>
                start
            </Button>
        );
    }

    if (!view.running && view.players.filter((p) => !p.sittingOut).length >= 2) {
        return (
            <Button variant="filled" size="md" className={bar} onClick={() => handleStartGame(socket)}>
                start
            </Button>
        );
    }

    return null;
}
