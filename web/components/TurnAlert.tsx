import { useContext, useEffect, useRef } from "react";
import { FiVolume2, FiVolumeX } from "react-icons/fi";
import { AppContext } from "../providers/AppStore";
import { ActionIcon } from "@mantine/core";

// TurnAlert is the header's mute, and gets your attention when it becomes your
// turn in a background tab by flashing the tab title. Sounds plays the chime.
export default function TurnAlert() {
    const { appState, dispatch } = useContext(AppContext);
    const muted = appState.prefs.muted;
    const myTurn = !!appState.view?.legal.turn;
    const wasMyTurn = useRef(false);

    useEffect(() => {
        const started = myTurn && !wasMyTurn.current;
        wasMyTurn.current = myTurn;
        if (!started || !document.hidden) return;

        const title = document.title;
        let id: ReturnType<typeof setInterval> | undefined;
        // wait a moment first, since a pre-action can take the turn straight away
        const wait = setTimeout(() => {
            let on = false;
            id = setInterval(() => {
                on = !on;
                document.title = on ? "● your turn" : title;
            }, 1000);
        }, 500);
        const stop = () => {
            clearTimeout(wait);
            clearInterval(id);
            document.title = title;
            document.removeEventListener("visibilitychange", stop);
        };
        document.addEventListener("visibilitychange", stop);
        return stop;
    }, [myTurn]);

    const toggle = () => dispatch({ type: "setPref", payload: { muted: !muted } });

    return (
        <ActionIcon
            size="sm"
            title={muted ? "unmute" : "mute"}
            aria-label="mute all sound"
            aria-pressed={muted}
            onClick={toggle}
        >
            {muted ? <FiVolumeX /> : <FiVolume2 />}
        </ActionIcon>
    );
}
