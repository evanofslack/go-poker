import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import InviteButton from "./InviteButton";

// hint says what to do next when the table can't play yet
function hint(seated: boolean, players: number, ready: number, running: boolean) {
    if (!seated && running) return "watching · pick an open seat to play next hand";
    if (players === 0) return "pick a seat to start · invite friends to join";
    if (!seated) return "pick an open seat to play";
    if (ready < 2) return running ? "waiting for another player to sit in" : "waiting for another player";
    return "";
}

export default function TableHint() {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (appState.connection !== "open") {
        return <p className="rounded-full bg-amber-400 px-4 py-1 text-sm text-zinc-900">reconnecting…</p>;
    }
    if (!game) return null;

    const seated = game.players.some((p) => p.uuid === appState.clientID);
    const ready = game.players.filter((p) => p.ready).length;
    const text = hint(seated, game.players.length, ready, game.running);
    if (!text) return null;

    return (
        <div className="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl bg-panel/80 px-3 py-1.5 text-center text-xs text-muted sm:text-sm">
            <span>{text}</span>
            {ready < 2 && <InviteButton />}
        </div>
    );
}
