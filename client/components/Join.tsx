import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { newPlayer } from "../actions/actions";

export default function Join() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [inputValue, setInputValue] = useState("");

    const handleSubmit = (e: any) => {
        e.preventDefault();
        dispatch({ type: "setUsername", payload: inputValue });
        if (socket) {
            newPlayer(socket, inputValue);
        }
        console.log(e.key);
    };

    const onKeyDown = (e: KeyboardEvent) => {
        if (e.key === "Enter") {
            handleSubmit(e);
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
        <div className="flex min-h-screen flex-col">
            <div className="flex flex-grow flex-col items-center justify-center ">
                <h1 className="mb-16 text-5xl font-semibold text-white">Poker</h1>
                <div>
                    <input
                        autoFocus
                        className="m-4 rounded-sm bg-neutral-600 p-1 text-white focus:outline-none"
                        id="input"
                        type="text"
                        value={inputValue}
                        placeholder="username"
                        onChange={handleChange}
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
