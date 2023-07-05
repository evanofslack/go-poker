import React, { useCallback, useContext, useEffect, useState, useRef } from "react";
import { useSocket } from "../../hooks/useSocket";
import ChatMessage from "./ChatMessage";
import { AppContext } from "../../providers/AppStore";
import useChatScroll from "../../hooks/useChatScroll";
import { FiSend } from "react-icons/fi";
import { sendMessage } from "../../actions/actions";

export default function Chat() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");
    const scrollRef = useChatScroll(appState.messages);
    const messageRef = useRef(null);

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
        <div className="flex h-full w-full grow flex-col overflow-auto">
            <div ref={scrollRef} className="mb-2 h-full overflow-auto bg-zinc-800 p-2">
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
            </div>
        </div>
    );
}
