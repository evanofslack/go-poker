import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { newPlayer } from "../actions/actions";

export default function Join() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");

    const handleSubmit = useCallback(
        (e) => {
            e.preventDefault();

            dispatch({ type: "setUsername", payload: inputValue });
            if (socket) {
                newPlayer(socket, inputValue);
            }
        },
        [inputValue]
    );

    const handleKeypress = useCallback((e: React.KeyboardEvent<HTMLDivElement>) => {
        if (e.key === "Enter") {
            handleSubmit(e);
        }
    }, []);

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
                        onKeyPress={handleKeypress}
                    />
                    <button
                        className="rounded-sm bg-green-800 px-4 py-1 text-white"
                        onClick={handleSubmit}
                    >
                        Join
                    </button>
                </div>
            </div>
            <Footer />
        </div>
    );
}
