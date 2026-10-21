import { useContext, useRef, useState } from "react";
import { FiChevronDown } from "react-icons/fi";
import { resetGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { useDismiss } from "../hooks/useDismiss";
import { AppContext } from "../providers/AppStore";
import Confirm from "./Confirm";

// HostMenu holds the controls only the host has. Reset is the only one so far.
export default function HostMenu() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [open, setOpen] = useState(false);
    const [confirming, setConfirming] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    useDismiss(ref, open, () => setOpen(false));

    if (!socket || appState.game?.host !== appState.clientID) {
        return null;
    }
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
                    className="absolute top-full left-0 z-40 mt-1 flex w-40 flex-col rounded-md border border-zinc-600 bg-panel py-1 text-fg shadow-xl"
                >
                    <button
                        role="menuitem"
                        className="px-3 py-1.5 text-left text-sm text-danger hover:bg-raised"
                        onClick={() => {
                            setOpen(false);
                            setConfirming(true);
                        }}
                    >
                        reset table
                    </button>
                </div>
            )}
            {confirming && (
                <Confirm
                    title="reset the table?"
                    message="everyone leaves their seat and the game stops."
                    action="reset"
                    onConfirm={() => {
                        resetGame(socket);
                        setConfirming(false);
                    }}
                    onCancel={() => setConfirming(false)}
                />
            )}
        </div>
    );
}
