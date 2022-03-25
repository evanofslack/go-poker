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
        <div className="App">
            <input id="input" type="text" value={inputValue} onChange={handleChange} />
            <button onClick={handleClick}>Send</button>
            <div>
                {AppStore.state.messages.map((message, index) => (
                    <ChatMessage key={index} name={message.name} message={message.message} />
                ))}
            </div>
        </div>
    );
}
