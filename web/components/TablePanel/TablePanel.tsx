import { useEffect, useRef, useState } from "react";
import classNames from "classnames";
import Players from "./Players";
import Stats from "./Stats";

type tab = "players" | "stats";
const tabKey = "tablePanelTab";

function savedTab(): tab {
    try {
        return localStorage.getItem(tabKey) === "stats" ? "stats" : "players";
    } catch {
        return "players";
    }
}

// TablePanel is a drawer over the left of the table with the players, ledger and
// stats. It isn't modal, so you can still act while it's open.
export default function TablePanel({ onClose }: { onClose: () => void }) {
    const [tab, setTab] = useState<tab>(savedTab);
    // callers pass onClose inline; a ref keeps rerenders from re-running the effect
    const close = useRef(onClose);
    close.current = onClose;

    useEffect(() => {
        // a dialog opened from the panel handles its own Escape
        const key = (e: KeyboardEvent) =>
            e.key === "Escape" && !document.querySelector('[aria-modal="true"]') && close.current();
        document.addEventListener("keydown", key);
        return () => document.removeEventListener("keydown", key);
    }, []);

    const pick = (t: tab) => {
        setTab(t);
        try {
            localStorage.setItem(tabKey, t);
        } catch {}
    };

    const tabStyle = (t: tab) =>
        classNames("rounded-md px-3 py-1 text-sm", tab === t ? "bg-surface text-fg" : "text-subtle");

    return (
        <div
            role="region"
            aria-label="table"
            className="absolute inset-y-2 right-2 left-2 z-40 flex flex-col rounded-xl bg-raised p-2 text-muted shadow-xl sm:right-auto sm:w-[28rem]"
        >
            <div className="mb-2 flex items-center gap-1">
                <button className={tabStyle("players")} onClick={() => pick("players")}>
                    players
                </button>
                <button className={tabStyle("stats")} onClick={() => pick("stats")}>
                    stats
                </button>
                <button
                    className="ml-auto px-2 text-subtle hover:text-fg"
                    title="close"
                    aria-label="close table panel"
                    onClick={onClose}
                >
                    ✕
                </button>
            </div>
            <div className="min-h-0 flex-1">
                {tab === "players" && <Players />}
                {tab === "stats" && <Stats />}
            </div>
        </div>
    );
}
