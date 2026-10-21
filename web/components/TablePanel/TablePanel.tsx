import { useContext, useEffect, useRef, useState } from "react";
import classNames from "classnames";
import { AppContext } from "../../providers/AppStore";
import Host from "./Host";
import Players from "./Players";
import Stats from "./Stats";

type tab = "players" | "stats" | "host";
const tabKey = "tablePanelTab";

// savedTab is the last tab opened. The host starts on the host tab.
function savedTab(host: boolean): tab {
    let saved = null;
    try {
        saved = localStorage.getItem(tabKey);
    } catch {}
    if (saved === "players" || saved === "stats" || saved === "host") return saved;
    return host ? "host" : "players";
}

// TablePanel is a drawer over the left of the table with the players, ledger,
// stats and the host's session controls. It isn't modal, so you can still act
// while it's open.
export default function TablePanel({ onClose }: { onClose: () => void }) {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    const host = !!game && game.host === appState.username && !game.ended;
    const [picked, setTab] = useState<tab>(() => savedTab(host));
    // the host tab goes when you stop being host or the session ends
    const tab = picked === "host" && !host ? "players" : picked;
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
                {host && (
                    <button className={tabStyle("host")} onClick={() => pick("host")}>
                        host
                    </button>
                )}
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
                {tab === "host" && <Host />}
            </div>
        </div>
    );
}
