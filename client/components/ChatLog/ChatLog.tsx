import React, { useState } from "react";
import { MdExpandLess } from "react-icons/md";
import { MdExpandMore } from "react-icons/md";
import classNames from "classnames";
import Chat from "./Chat";
import Log from "./Log";

export default function ChatLog() {
    const [expand, setExpand] = useState(false);
    const [showChat, setShowChat] = useState(true);

    function chatHeight(expand: boolean) {
        return classNames(
            {
                "h-72": expand,
                "h-36": !expand,
            },
            "flex w-96 flex-col items-start justify-between rounded-tr-lg bg-zinc-700 p-3 text-neutral-400"
        );
    }

    return (
        <div className={chatHeight(expand)}>
            {showChat && <Chat />}
            {!showChat && <Log />}
            <button className="absolute top-0 left-0" onClick={() => setShowChat(!showChat)}>
                Show {showChat ? "logs" : "chat"}
            </button>
            {expand ? (
                <button
                    className="absolute top-0 right-0 pt-3 pr-7"
                    onClick={() => setExpand(!expand)}
                >
                    <MdExpandMore size="1.7rem" />
                </button>
            ) : (
                <button
                    className="absolute top-0 right-0 pt-3 pr-7"
                    onClick={() => setExpand(!expand)}
                >
                    <MdExpandLess size="1.7rem" />
                </button>
            )}
        </div>
    );
}
