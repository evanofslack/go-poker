import { useContext, useRef, useState } from "react";
import { FiChevronDown } from "react-icons/fi";
import { sessionAction } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { useDismiss } from "../hooks/useDismiss";
import { AppContext } from "../providers/AppStore";
import Confirm from "./Confirm";

// HostMenu holds the host's controls for the whole session. Controls for one
// player are on their row in the players panel.
export default function HostMenu() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [open, setOpen] = useState(false);
    const [confirming, setConfirming] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    useDismiss(ref, open, () => setOpen(false));
    const game = appState.game;

    if (!socket || !game || game.host !== appState.username || game.ended) {
        return null;
    }
    const midHand = game.stage !== 1;
    const item = "px-3 py-1.5 text-left text-sm hover:bg-raised";
    return (
        <div ref={ref} className="relative">
            <button
                className="flex items-center gap-1 rounded-md px-2 py-1 text-sm text-subtle hover:text-fg"
                aria-haspopup="menu"
                aria-expanded={open}
                onClick={() => setOpen(!open)}
            >
                host <FiChevronDown />
            </button>
            {open && (
                <div
                    role="menu"
                    className="absolute top-full left-0 z-40 mt-1 flex w-44 flex-col rounded-md border border-line bg-panel py-1 text-fg shadow-xl"
                >
                    <button
                        role="menuitem"
                        className={item}
                        onClick={() => {
                            setOpen(false);
                            sessionAction(socket, game.paused ? "resume-game" : "pause-game");
                        }}
                    >
                        {game.paused ? "resume" : "pause"}
                    </button>
                    {!game.ending && (
                        <button
                            role="menuitem"
                            className={item + " text-danger"}
                            onClick={() => {
                                setOpen(false);
                                setConfirming(true);
                            }}
                        >
                            end session
                        </button>
                    )}
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
