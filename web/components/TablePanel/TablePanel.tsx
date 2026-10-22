import { useContext, useEffect, useRef, useState } from "react";
import classNames from "classnames";
import { AppContext } from "../../providers/AppStore";
import History from "./History";
import Host from "./Host";
import Players from "./Players";
import Settings from "./Settings";
import Stats from "./Stats";

type tab = "players" | "hands" | "stats" | "settings" | "host";
const tabs: tab[] = ["players", "hands", "stats", "settings", "host"];
const tabKey = "tablePanelTab";

// savedTab is the last tab opened. The host starts on the host tab.
function savedTab(host: boolean): tab {
    let saved = null;
    try {
        saved = localStorage.getItem(tabKey);
    } catch {}
    if (tabs.includes(saved as tab)) return saved as tab;
    return host ? "host" : "players";
}

// TablePanel is a drawer over the left of the table with the players, ledger,
// stats, table settings and the host's session controls. It isn't modal, so you can still act
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
        classNames("shrink-0 rounded-md px-2 py-1 text-sm sm:px-3", tab === t ? "bg-surface text-fg" : "text-subtle");

    return (
        <div
            role="region"
            aria-label="table"
            className="absolute inset-y-2 right-2 left-2 z-40 flex flex-col rounded-xl bg-raised p-2 text-muted shadow-float ring-1 ring-black/20 sm:right-auto sm:w-[28rem]"
        >
            <div className="mb-2 flex items-center gap-1">
                <div className="flex min-w-0 gap-1 overflow-x-auto">
                    <button className={tabStyle("players")} onClick={() => pick("players")}>
                        players
                    </button>
                    <button className={tabStyle("hands")} onClick={() => pick("hands")}>
                        hands
                    </button>
                    <button className={tabStyle("stats")} onClick={() => pick("stats")}>
                        stats
                    </button>
                    <button className={tabStyle("settings")} onClick={() => pick("settings")}>
                        settings
                    </button>
                    {host && (
                        <button className={tabStyle("host")} onClick={() => pick("host")}>
                            host
                        </button>
                    )}
                </div>
                <button
                    className="ml-auto shrink-0 px-2 text-subtle hover:text-fg"
                    title="close"
                    aria-label="close table panel"
                    onClick={onClose}
                >
                    ✕
                </button>
            </div>
            <div className="min-h-0 flex-1">
                {tab === "players" && <Players />}
                {tab === "hands" && <History />}
                {tab === "stats" && <Stats />}
                {tab === "settings" && <Settings />}
                {tab === "host" && <Host />}
            </div>
        </div>
    );
}
