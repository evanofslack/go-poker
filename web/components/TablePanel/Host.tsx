import { useContext, useState } from "react";
import classNames from "classnames";
import { cancelStandUp, sessionAction, startStandUp } from "../../actions/actions";
import { standUpPayouts } from "../../lib/settings";
import { useSocket } from "../../hooks/useSocket";
import { AppContext } from "../../providers/AppStore";
import { Button } from "@mantine/core";
import Confirm from "../Confirm";
import Section from "../Section";
import Segmented from "../Segmented";

// Host holds the host's controls for the whole session. Controls for one
// player are on their row in the players tab.
export default function Host() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [confirming, setConfirming] = useState(false);
    const [payoutBB, setPayoutBB] = useState(2);
    const game = appState.view;
    if (!socket || !game) return null;
    const midHand = game.street !== "";

    let status = "running";
    if (game.ended) status = "session over";
    else if (game.ending) status = "ending after this hand";
    else if (game.paused) status = midHand ? "pausing after this hand" : "paused";

    return (
        <div className="flex h-full flex-col gap-5 overflow-auto bg-surface p-3 text-sm">
            <div className="flex items-center gap-2" aria-label="game status">
                <span
                    className={classNames(
                        "size-2 shrink-0 rounded-full",
                        status === "running" ? "bg-gain" : game.paused ? "bg-warn" : "bg-subtle"
                    )}
                />
                <span className="text-fg">{status}</span>
                <Button
                    size="xs"
                    className="ml-auto"
                    onClick={() => sessionAction(socket, game.paused ? "resume-game" : "pause-game")}
                >
                    {game.paused ? "resume" : "pause"}
                </Button>
            </div>
            {!game.ended && <StandUp payoutBB={payoutBB} setPayoutBB={setPayoutBB} />}
            {!game.ending && (
                <Button variant="outline" color="red" className="mt-auto self-start" onClick={() => setConfirming(true)}>
                    end session
                </Button>
            )}
            {confirming && (
                <Confirm
                    title="end the session?"
                    message={
                        (midHand ? "after this hand, " : "") +
                        "no more hands are dealt and the ledger is final. the table closes an hour later."
                    }
                    action="end session"
                    onConfirm={() => {
                        sessionAction(socket, "end-session");
                        setConfirming(false);
                    }}
                    onCancel={() => setConfirming(false)}
                />
            )}
        </div>
    );
}

const rules =
    "everyone dealt in stands until they win a pot and show. the last one standing pays each player who sat down. rounds begin on the next hand.";

// Names lists players as small pills
function Names({ label, names }: { label: string; names: string[] }) {
    return (
        <div className="flex flex-wrap items-center gap-1">
            <span className="mr-1 text-xs text-subtle">{label}</span>
            {names.map((n) => (
                <span key={n} className="rounded bg-badge px-1.5 text-fg">
                    {n}
                </span>
            ))}
        </div>
    );
}

// StandUp starts and cancels stand-up rounds: everyone dealt into the first
// hand stands until they win a pot and show, and the last one standing pays
// everyone who sat down.
function StandUp({ payoutBB, setPayoutBB }: { payoutBB: number; setPayoutBB: (n: number) => void }) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const game = appState.view;
    if (!socket || !game) return null;
    const u = game.standUp;

    if (u) {
        const sat = u.sat ?? [];
        const standing = (u.players ?? []).filter((n) => !sat.includes(n));
        return (
            <Section title="stand-up" hint={rules}>
                <p className="flex flex-wrap items-center gap-2 text-fg" aria-label="stand-up round">
                    {u.payout.toLocaleString("en")} each
                    <span className="rounded bg-badge px-1.5 text-xs text-muted">
                        {u.pending ? "starts with the next hand" : "in play"}
                    </span>
                </p>
                {!u.pending && (
                    <>
                        <Names label="still standing" names={standing} />
                        {sat.length > 0 && <Names label="sat down" names={sat} />}
                    </>
                )}
                <Button className="self-start" onClick={() => cancelStandUp(socket)}>
                    cancel round
                </Button>
            </Section>
        );
    }
    return (
        <Section title="stand-up" hint={rules}>
            <Segmented
                legend="payout"
                name="standUpPayout"
                options={standUpPayouts.map((n) => ({
                    label: (
                        <>
                            <span>{n} bb</span>{" "}
                            <span className="font-normal opacity-60">
                                {(n * game.settings.bb).toLocaleString("en")}
                            </span>
                        </>
                    ),
                    value: n,
                }))}
                value={payoutBB}
                onChange={setPayoutBB}
            />
            <Button
                variant="filled"
                className="self-start"
                onClick={() => startStandUp(socket, payoutBB * game.settings.bb)}
            >
                start stand-up
            </Button>
        </Section>
    );
}
