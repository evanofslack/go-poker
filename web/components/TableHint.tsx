import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { Game } from "../interfaces/index";
import InviteButton from "./InviteButton";

// sessionHint says what the host did to the session, which matters to everyone
function sessionHint(game: Game) {
    if (game.ended) {
        const minutes = Math.max(1, Math.ceil(game.closesIn / 60000));
        return `the session is over · the ledger is final · table closes in ${minutes}m`;
    }
    if (game.ending) return "last hand · the session ends after it";
    if (game.paused) return game.stage === 1 ? "paused by the host" : "the host pauses the game after this hand";
    return "";
}

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
    const mine = game.buyRequests.find((r) => r.name === appState.username);
    const session =
        sessionHint(game) || (mine ? `waiting for the host to ok your buy-in of ${mine.amount}` : "");
    const text = session || hint(seated, game.players.length, ready, game.running);
    if (!text) return null;

    return (
        <div className="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl bg-panel/80 px-3 py-1.5 text-center text-xs text-muted sm:text-sm">
            <span>{text}</span>
            {ready < 2 && !session && <InviteButton />}
        </div>
    );
}
