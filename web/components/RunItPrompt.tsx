import { useContext } from "react";
import classNames from "classnames";
import { runIt } from "../actions/actions";
import { useSecondsLeft } from "../hooks/useSecondsLeft";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Button from "./Button";

// RunItPrompt asks the players still in an all-in hand how many times to run
// the board out. Everyone else sees who has answered.
export default function RunItPrompt() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const view = appState.view;
    const left = useSecondsLeft(view?.runVoteEndsIn ?? 0);
    const vote = view?.runVote;
    if (!socket || !view || !vote || appState.connection !== "open") return null;

    const picks = vote.picks ?? {};
    if (!vote.seats.includes(view.you)) {
        const waiting = vote.seats
            .filter((n) => !(n in picks))
            .map((n) => view.players.find((p) => p.seat === n)?.name);
        return (
            <p className="self-center text-sm text-subtle" data-run-wait>
                {waiting.join(" and ")} {waiting.length > 1 ? "are" : "is"} deciding how many times to run it{" "}
                <span className="tabular-nums">{left}s</span>
            </p>
        );
    }
    const mine = picks[view.you];
    return (
        <div role="group" aria-label="run it" className="flex items-center gap-2" data-run-ask>
            <span className="text-sm text-subtle">run it</span>
            {Array.from({ length: vote.max }, (_, i) => i + 1).map((n) => (
                <Button
                    key={n}
                    variant={mine === n ? "primary" : undefined}
                    className={classNames("px-2 sm:px-4 sm:text-xl", mine && mine !== n && "opacity-60")}
                    aria-pressed={mine === n}
                    onClick={() => runIt(socket, n)}
                >
                    {n === 1 ? "once" : `${n}×`}
                </Button>
            ))}
            <span className="text-sm text-subtle tabular-nums">{left}s</span>
        </div>
    );
}
