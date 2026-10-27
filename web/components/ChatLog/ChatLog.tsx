import { useContext, useEffect, useState } from "react";
import classNames from "classnames";
import { AppContext } from "../../providers/AppStore";
import Chat from "./Chat";
import Log from "./Log";
import { ActionIcon, Button } from "@mantine/core";
import { FiX } from "react-icons/fi";

// ChatLog is a toggle that opens the chat and game log above the footer
export default function ChatLog() {
    const { appState } = useContext(AppContext);
    const [open, setOpen] = useState(false);
    const [tab, setTab] = useState<"chat" | "log">("chat");
    const [seen, setSeen] = useState(0);

    const count = appState.messages.length;
    useEffect(() => {
        if (open) setSeen(count);
    }, [open, count]);
    const unread = open ? 0 : count - seen;

    const tabStyle = (t: string) =>
        classNames(
            "rounded-md px-3 py-1 text-sm",
            tab === t ? "bg-surface text-fg" : "text-subtle"
        );

    return (
        <>
            <Button
                className="shrink-0 overflow-visible"
                aria-label={unread > 0 ? `chat, ${unread} unread` : "chat"}
                aria-expanded={open}
                onClick={() => setOpen(!open)}
            >
                chat
                {unread > 0 && (
                    <span className="absolute -top-1.5 -right-1.5 rounded-full bg-accent px-1.5 text-xs text-on-accent">
                        {unread}
                    </span>
                )}
            </Button>
            {open && (
                <div className="absolute right-2 bottom-full left-2 z-40 flex h-72 flex-col rounded-xl bg-raised p-2 text-muted shadow-float ring-1 ring-black/20 sm:right-auto sm:left-4 sm:w-96">
                    <div className="mb-2 flex items-center gap-1">
                        <button className={tabStyle("chat")} onClick={() => setTab("chat")}>
                            chat
                        </button>
                        <button className={tabStyle("log")} onClick={() => setTab("log")}>
                            log
                        </button>
                        <ActionIcon className="ml-auto" title="close" aria-label="close chat" onClick={() => setOpen(false)}>
                            <FiX />
                        </ActionIcon>
                    </div>
                    <div className="min-h-0 flex-1">
                        {tab === "chat" && <Chat />}
                        {tab === "log" && <Log />}
                    </div>
                </div>
            )}
        </>
    );
}
