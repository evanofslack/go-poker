import { useContext, useState } from "react";
import { cancelStandUp, sessionAction, startStandUp } from "../../actions/actions";
import { standUpPayouts } from "../../lib/settings";
import { useSocket } from "../../hooks/useSocket";
import { AppContext } from "../../providers/AppStore";
import Button from "../Button";
import Confirm from "../Confirm";
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

    return (
        <div className="flex h-full flex-col gap-3 overflow-auto bg-surface p-3 text-sm">
            <div className="flex items-center gap-3">
                <Button
                    variant="primary"
                    className="px-4 py-1.5"
                    onClick={() => sessionAction(socket, game.paused ? "resume-game" : "pause-game")}
                >
                    {game.paused ? "resume" : "pause"}
                </Button>
                <span className="text-subtle">
                    {game.paused
                        ? midHand
                            ? "pausing after this hand"
                            : "the game is paused"
                        : "no new hands are dealt until you resume"}
                </span>
            </div>
            {!game.ended && <StandUp payoutBB={payoutBB} setPayoutBB={setPayoutBB} />}
            {!game.ending && (
                <div className="flex items-center gap-3 border-t border-raised pt-3">
                    <Button variant="danger" className="px-4 py-1.5" onClick={() => setConfirming(true)}>
                        end session
                    </Button>
                    <span className="text-subtle">settle up and close the table</span>
                </div>
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
            <div className="flex flex-col gap-2 border-t border-raised pt-3" aria-label="stand-up round">
                <p className="text-fg">
                    stand-up round, {u.payout.toLocaleString("en")} each
                    {u.pending && <span className="text-subtle"> · starts with the next hand</span>}
                </p>
                {!u.pending && (
                    <p className="text-subtle">
                        standing: <span className="text-fg">{standing.join(", ")}</span>
                        {sat.length > 0 && (
                            <>
                                {" "}
                                · sat down: <span className="text-fg">{sat.join(", ")}</span>
                            </>
                        )}
                    </p>
                )}
                <div>
                    <Button className="px-4 py-1.5" onClick={() => cancelStandUp(socket)}>
                        cancel round
                    </Button>
                </div>
            </div>
        );
    }
    return (
        <div className="flex flex-col gap-2 border-t border-raised pt-3">
            <Segmented
                legend="stand-up: the last one standing pays each player who sat down"
                name="standUpPayout"
                options={standUpPayouts.map((n) => ({ label: `${n} bb`, value: n }))}
                value={payoutBB}
                onChange={setPayoutBB}
            />
            <div className="flex items-center gap-3">
                <Button
                    variant="primary"
                    className="px-4 py-1.5"
                    onClick={() => startStandUp(socket, payoutBB * game.settings.bb)}
                >
                    start stand-up
                </Button>
                <span className="text-subtle">from the next hand</span>
            </div>
        </div>
    );
}
