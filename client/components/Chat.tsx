import React, { useCallback, useContext, useEffect, useState, useRef } from "react";
import { useSocket } from "../hooks/useSocket";
import { Message } from "../interfaces/index";
import ChatMessage from "./ChatMessage";
import { AppContext } from "../providers/AppStore";
import useChatScroll from "../hooks/useChatScroll";
import { FiSend } from "react-icons/fi";
import { MdExpandLess } from "react-icons/md";
import { MdExpandMore } from "react-icons/md";
import { sendMessage } from "../actions/actions";
import classNames from "classnames";

export default function Chat() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");
    const scrollRef = useChatScroll(appState.messages);
    const [expand, setExpand] = useState(false);
    const messageRef = useRef(null);

    function chatHeight(expand: boolean) {
        return classNames(
            {
                "h-72": expand,
                "h-36": !expand,
            },
            "flex w-96 flex-col items-start justify-between rounded-tr-lg bg-zinc-700 p-3 text-neutral-400"
        );
    }

    const handleClick = useCallback(
        (e) => {
            e.preventDefault();
            if (socket && appState.username && inputValue != "") {
                sendMessage(socket, appState.username, inputValue);
            }
            setInputValue("");
        },
        [inputValue]
    );

    const onKeyDown = (e: KeyboardEvent) => {
        if (document.activeElement == messageRef.current && inputValue != "") {
            if (e.key === "Enter") {
                handleClick(e);
            }
        }
    };

    useEffect(() => {
        document.addEventListener("keydown", onKeyDown);
        return () => {
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [onKeyDown]);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);
    }, []);

    return (
        <div className={chatHeight(expand)}>
            <div ref={scrollRef} className="mb-2 h-full w-full overflow-auto bg-zinc-800 p-2">
                {appState.messages.map((message, index) => (
                    <ChatMessage
                        key={index}
                        name={message.name}
                        message={message.message}
                        timestamp={message.timestamp}
                    />
                ))}
            </div>
            <div className="flex w-full flex-row justify-between text-neutral-400">
                <input
                    className="w-full bg-zinc-800 p-2 text-neutral-400 placeholder-neutral-600"
                    id="input"
                    type="text"
                    value={inputValue}
                    placeholder="say something..."
                    onChange={handleChange}
                    ref={messageRef}
                ></input>
                <button className=" bg-zinc-800 px-4 text-neutral-400" onClick={handleClick}>
                    <FiSend />
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
        </div>
    );
}
