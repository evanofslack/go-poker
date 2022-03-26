import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";

export default function Join() {
    const socket = useSocket();
    const AppStore = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");

    const handleClick = useCallback(
        (e) => {
            e.preventDefault();

            AppStore.dispatch({ type: "setUsername", payload: inputValue });

            socket?.send(
                JSON.stringify({
                    action: "new-player",
                    params: {
                        username: inputValue,
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
        <div className="flex min-h-screen flex-col">
            <div className="flex flex-grow flex-col items-center justify-center ">
                <h1 className="mb-16 text-5xl font-semibold text-white">PokerGo</h1>
                <div>
                    <input
                        className="m-4 rounded-sm bg-gray-600 p-1 text-white"
                        id="input"
                        type="text"
                        value={inputValue}
                        placeholder="username"
                        onChange={handleChange}
                    />
                    <button
                        className="rounded-sm bg-green-800 px-4 py-1 text-white"
                        onClick={handleClick}
                    >
                        Join
                    </button>
                </div>
            </div>
            <Footer />
        </div>
    );
}
