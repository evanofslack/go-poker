import { useContext, useEffect, useRef, useState } from "react";
import { AppContext } from "../../providers/AppStore";
import History from "./History";
import Host from "./Host";
import Players from "./Players";
import Settings from "./Settings";
import Options from "./Options";
import Stats from "./Stats";
import { ActionIcon, Tabs } from "@mantine/core";
import { FiX } from "react-icons/fi";

type tab = "players" | "hands" | "stats" | "settings" | "options" | "host";
const tabs: tab[] = ["players", "hands", "stats", "settings", "options", "host"];
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
// stats, table settings, your own options and the host's session controls. It isn't modal, so you can still act
// while it's open.
export default function TablePanel({ onClose }: { onClose: () => void }) {
    const { appState } = useContext(AppContext);
    const game = appState.view;
    const host = !!game && game.host === appState.username && !game.ended;
    const [picked, setTab] = useState<tab>(() => savedTab(host));
    // the host tab goes when you stop being host or the session ends
    const tab = picked === "host" && !host ? "players" : picked;
    // callers pass onClose inline; a ref keeps rerenders from re-running the effect
    const close = useRef(onClose);
    close.current = onClose;

    useEffect(() => {
        // a dialog or popover opened from the panel handles its own Escape. Listen in the
        // capture phase, before it closes, so it's still there to see.
        const open = '[aria-modal="true"], [aria-haspopup][aria-expanded="true"]';
        const key = (e: KeyboardEvent) => e.key === "Escape" && !document.querySelector(open) && close.current();
        window.addEventListener("keydown", key, true);
        return () => window.removeEventListener("keydown", key, true);
    }, []);

    const pick = (t: tab) => {
        setTab(t);
        try {
            localStorage.setItem(tabKey, t);
        } catch {}
    };

    return (
        <div
            role="region"
            aria-label="table"
            className="absolute inset-y-2 right-2 left-2 z-40 flex flex-col rounded-xl bg-raised p-2 text-muted shadow-float ring-1 ring-black/20 sm:right-auto sm:w-[28rem]"
        >
            <Tabs
                value={tab}
                onChange={(t) => t && pick(t as tab)}
                keepMounted={false}
                className="flex min-h-0 flex-1 flex-col"
            >
                <div className="mb-2 flex items-center gap-1">
                    <Tabs.List className="min-w-0 flex-1 flex-nowrap overflow-x-auto">
                        {tabs
                            .filter((t) => t !== "host" || host)
                            .map((t) => (
                                <Tabs.Tab key={t} value={t} className="shrink-0 px-2 sm:px-3">
                                    {t}
                                </Tabs.Tab>
                            ))}
                    </Tabs.List>
                    <ActionIcon className="shrink-0" title="close" aria-label="close table panel" onClick={onClose}>
                        <FiX />
                    </ActionIcon>
                </div>
                <Tabs.Panel value="players" className="min-h-0 flex-1">
                    <Players />
                </Tabs.Panel>
                <Tabs.Panel value="hands" className="min-h-0 flex-1">
                    <History />
                </Tabs.Panel>
                <Tabs.Panel value="stats" className="min-h-0 flex-1">
                    <Stats />
                </Tabs.Panel>
                <Tabs.Panel value="settings" className="min-h-0 flex-1">
                    <Settings />
                </Tabs.Panel>
                <Tabs.Panel value="options" className="min-h-0 flex-1">
                    <Options />
                </Tabs.Panel>
                <Tabs.Panel value="host" className="min-h-0 flex-1">
                    <Host />
                </Tabs.Panel>
            </Tabs>
        </div>
    );
}
