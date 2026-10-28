import { useContext, useEffect, useRef, useState } from "react";
import { FiVolume2, FiVolumeX } from "react-icons/fi";
import { AppContext } from "../providers/AppStore";
import { ActionIcon } from "@mantine/core";

function loadMuted() {
    try {
        return localStorage.getItem("muted") === "1";
    } catch {
        return false;
    }
}

function beep() {
    try {
        const ctx = new AudioContext();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.frequency.value = 660;
        gain.gain.setValueAtTime(0.15, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.3);
        osc.connect(gain).connect(ctx.destination);
        osc.start();
        osc.stop(ctx.currentTime + 0.3);
        osc.onended = () => ctx.close();
    } catch {}
}

// TurnAlert gets your attention when it becomes your turn in a background tab:
// it flashes the tab title and, unless muted, plays a short sound.
export default function TurnAlert() {
    const { appState } = useContext(AppContext);
    const [muted, setMuted] = useState(loadMuted);
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
            if (!muted) beep();
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
    }, [myTurn, muted]);

    const toggle = () => {
        setMuted(!muted);
        try {
            localStorage.setItem("muted", muted ? "0" : "1");
        } catch {}
    };

    return (
        <ActionIcon
            size="sm"
            title={muted ? "unmute" : "mute"}
            aria-label="mute turn sound"
            aria-pressed={muted}
            onClick={toggle}
        >
            {muted ? <FiVolumeX /> : <FiVolume2 />}
        </ActionIcon>
    );
}
