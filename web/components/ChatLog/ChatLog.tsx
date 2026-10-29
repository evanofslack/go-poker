import { useContext, useEffect, useState } from "react";
import { AppContext } from "../../providers/AppStore";
import { SystemName } from "../../interfaces/table";
import Chat from "./Chat";
import Log from "./Log";
import { ActionIcon, Button, Tabs } from "@mantine/core";
import { FiX } from "react-icons/fi";

// ChatLog is a toggle that opens the chat and game log above the footer
export default function ChatLog() {
    const { appState } = useContext(AppContext);
    const [open, setOpen] = useState(false);
    const [tab, setTab] = useState<"chat" | "log">("chat");
    const [seen, setSeen] = useState(0);

    // your own messages, and the line saying you joined, are never news to you
    const me = appState.username;
    const count = appState.messages.filter(
        (m) =>
            m.name !== me &&
            !(m.name === SystemName && (m.message === `${me} has joined` || m.message.startsWith(`${me} rejoined`)))
    ).length;
    useEffect(() => {
        if (open) setSeen(count);
    }, [open, count]);
    const unread = open ? 0 : count - seen;

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
                    <Tabs
                        value={tab}
                        onChange={(t) => t && setTab(t as "chat" | "log")}
                        keepMounted={false}
                        className="flex min-h-0 flex-1 flex-col"
                    >
                        <div className="mb-2 flex items-center gap-1">
                            <Tabs.List className="flex-1">
                                <Tabs.Tab value="chat">chat</Tabs.Tab>
                                <Tabs.Tab value="log">log</Tabs.Tab>
                            </Tabs.List>
                            <ActionIcon title="close" aria-label="close chat" onClick={() => setOpen(false)}>
                                <FiX />
                            </ActionIcon>
                        </div>
                        <Tabs.Panel value="chat" className="min-h-0 flex-1">
                            <Chat />
                        </Tabs.Panel>
                        <Tabs.Panel value="log" className="min-h-0 flex-1">
                            <Log />
                        </Tabs.Panel>
                    </Tabs>
                </div>
            )}
        </>
    );
}
