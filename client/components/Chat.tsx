import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { Message } from "../interfaces/index";
import ChatMessage from "./ChatMessage";
import { AppContext } from "../providers/AppStore";

export default function Chat() {
    const socket = useSocket();
    const AppStore = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");

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
        <div className="flex h-64 w-96 flex-col items-start justify-start bg-gray-600 p-4 text-white">
            <div className="flex flex-row justify-between">
                <input
                    className="mr-8 bg-gray-800 p-2"
                    id="input"
                    type="text"
                    value={inputValue}
                    placeholder="say something..."
                    onChange={handleChange}
                />
                <button className="bg-black px-4" onClick={handleClick}>
                    Send
                </button>
            </div>
            <div className="mt-2 w-full bg-gray-800 p-2">
                {AppStore.state.messages.map((message, index) => (
                    <ChatMessage
                        key={index}
                        name={message.name}
                        message={message.message}
                        timestamp={message.timestamp}
                    />
                ))}
            </div>
        </div>
    );
}
