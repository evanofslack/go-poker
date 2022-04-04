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
        <div className="flex h-48 w-96 flex-col items-start justify-between bg-gray-600 p-4 text-white">
            <div ref={scrollRef} className="mb-2 w-full overflow-auto bg-gray-800 p-2">
                {appState.messages.map((message, index) => (
                    <ChatMessage
                        key={index}
                        name={message.name}
                        message={message.message}
                        timestamp={message.timestamp}
                    />
                ))}
            </div>
            <div className="flex w-full flex-row justify-between">
                <input
                    className="w-full bg-gray-800 p-2 "
                    id="input"
                    type="text"
                    value={inputValue}
                    placeholder="say something..."
                    onChange={handleChange}
                />
                <button className=" px-4" onClick={handleClick}>
                    <FiSend />
                </button>
            </div>
        </div>
    );
}
