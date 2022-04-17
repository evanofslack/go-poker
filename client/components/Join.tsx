import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { newPlayer, joinTable } from "../actions/actions";

export default function Join() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [username, setUsername] = useState("");
    const [tablename, setTablename] = useState("");

    const handleSubmit = (e: any) => {
        e.preventDefault();
        dispatch({ type: "setUsername", payload: username });
        if (socket) {
            joinTable(socket, tablename);
            newPlayer(socket, username);
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

    const handleUsername = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setUsername(e.target.value);
    }, []);

    const handleTablename = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setTablename(e.target.value);
    }, []);

    return (
        <div className="flex min-h-screen flex-col">
            <div className="flex flex-grow flex-col items-center justify-center ">
                <h1 className="mb-16 text-5xl font-semibold text-white">go poker</h1>
                <div>
                    <input
                        autoFocus
                        className="m-4 rounded-sm bg-neutral-600 p-1 text-white focus:outline-none"
                        id="username"
                        type="text"
                        value={username}
                        placeholder="username"
                        onChange={handleUsername}
                    />
                    <input
                        autoFocus
                        className="mr-4 w-24 rounded-sm bg-neutral-600 p-1 text-white focus:outline-none"
                        id="tablename"
                        type="text"
                        value={tablename}
                        placeholder="tablename"
                        onChange={handleTablename}
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
