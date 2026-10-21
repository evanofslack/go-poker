import { useContext, useState } from "react";
import { sessionAction } from "../../actions/actions";
import { useSocket } from "../../hooks/useSocket";
import { AppContext } from "../../providers/AppStore";
import Button from "../Button";
import Confirm from "../Confirm";

// Host holds the host's controls for the whole session. Controls for one
// player are on their row in the players tab.
export default function Host() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [confirming, setConfirming] = useState(false);
    const game = appState.game;
    if (!socket || !game) return null;
    const midHand = game.stage !== 1;

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
