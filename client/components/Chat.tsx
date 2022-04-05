import React, { useCallback, useContext, useEffect, useState, useRef } from "react";
import { useSocket } from "../hooks/useSocket";
import { Message } from "../interfaces/index";
import ChatMessage from "./ChatMessage";
import { AppContext } from "../providers/AppStore";
import useChatScroll from "../hooks/useChatScroll";
import { FiSend } from "react-icons/fi";
import { sendMessage } from "../actions/actions";

export default function Chat() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");
    const scrollRef = useChatScroll(appState.messages);

    const handleClick = useCallback(
        (e) => {
            e.preventDefault();
            setInputValue("");

            if (socket && appState.username) {
                sendMessage(socket, appState.username, inputValue);
            }
        },
        [inputValue]
    );

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);
    }, []);

    return (
        <div className="flex h-44 w-96 flex-col items-start justify-between rounded-tr-lg bg-zinc-700 p-3 text-neutral-400">
            <div ref={scrollRef} className="mb-2 w-full overflow-auto bg-zinc-800 p-2">
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
                    className="w-full bg-zinc-800 p-2 text-red-500"
                    id="input"
                    type="text"
                    value={inputValue}
                    placeholder="say something..."
                    onChange={handleChange}
                ></input>
                <button className=" bg-zinc-800 px-4 text-neutral-400" onClick={handleClick}>
                    <FiSend />
                </button>
            </div>
        </div>
    );
}
