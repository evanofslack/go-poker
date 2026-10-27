import { useContext } from "react";
import { FiX } from "react-icons/fi";
import { cancelStandUp, startStandUp } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { ActionIcon, Button } from "@mantine/core";
import { bar } from "../lib/theme";

// StandUpAgain lets the host run another stand-up round for the same payout
// once one ends, without stopping the game to ask.
export default function StandUpAgain() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!socket || !view || view.ended || view.standUp || !view.lastStandUp || view.host !== appState.username) {
        return null;
    }
    return (
        <div className="flex items-center gap-1">
            <Button size="md" className={bar} onClick={() => startStandUp(socket, view.lastStandUp)}>
                stand-up again
            </Button>
            <ActionIcon aria-label="no more stand-up" title="no more stand-up" onClick={() => cancelStandUp(socket)}>
                <FiX />
            </ActionIcon>
        </div>
    );
}
