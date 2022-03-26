import React, { useCallback, useContext, useEffect, useState, useRef } from "react";
import { useSocket } from "../hooks/useSocket";
import { Message } from "../interfaces/index";
import ChatMessage from "./ChatMessage";
import { AppContext } from "../providers/AppStore";
import useChatScroll from "../hooks/useChatScroll";
import { FiSend } from "react-icons/fi";

export default function Chat() {
    const socket = useSocket();
    const AppStore = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");
    const scrollRef = useChatScroll(AppStore.state.messages);

    useEffect(() => {
        if (socket) {
            socket.onopen = () => {
                console.log("Connected");
            };

            socket.onmessage = (e) => {
                let data = JSON.parse(e.data);
                let newMessage: Message = {
                    name: data.params.username,
                    message: data.params.message,
                    timestamp: data.params.timestamp,
                };
                AppStore.dispatch({ type: "addMessage", payload: newMessage });
            };
        }

        return () => {
            socket?.close();
        };
    }, []);

    const handleClick = useCallback(
        (e) => {
            e.preventDefault();
            setInputValue("");

            socket?.send(
                JSON.stringify({
                    action: "send-message",
                    params: {
                        username: AppStore.state.username,
                        message: inputValue,
                    },
                })
            );
        },
        [inputValue]
    );

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);
    }, []);

    return (
        <div className="flex h-64 w-96 flex-col items-start justify-between bg-gray-600 p-4 text-white">
            <div ref={scrollRef} className="mb-2 w-full overflow-auto bg-gray-800 p-2">
                {AppStore.state.messages.map((message, index) => (
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
