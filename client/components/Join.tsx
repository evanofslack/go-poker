import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { newPlayer, joinTable } from "../actions/actions";

function randomTableGen(length: number): string {
    let chars = "abcdefghijklmnopqrstuvwxyz0123456789";
    let result: string = "";
    for (let i = 0; i < length; i++) {
        result += chars[Math.floor(Math.random() * chars.length)];
    }
    return result;
}

export default function Join() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [username, setUsername] = useState("");
    const [tablename, setTablename] = useState("");
    let invalid = username == "" || tablename == "";

    const handleSubmit = (e: any) => {
        e.preventDefault();
        dispatch({ type: "setUsername", payload: username });
        dispatch({ type: "setTablename", payload: tablename });
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
                <div className="flex flex-row">
                    <input
                        autoFocus
                        className="m-2 rounded-sm bg-neutral-700 py-2 pl-4 text-white focus:outline-none"
                        id="username"
                        type="text"
                        value={username}
                        placeholder="name"
                        onChange={handleUsername}
                    />
                    <div className=" my-2 mr-2 flex flex-row rounded-sm bg-neutral-700 text-white">
                        <input
                            autoFocus
                            className="w-28 rounded-sm bg-neutral-700 py-2 pl-4 text-white focus:outline-none"
                            id="tablename"
                            type="text"
                            value={tablename}
                            placeholder="table"
                            onChange={handleTablename}
                        />
                        <button
                            onClick={() => setTablename(randomTableGen(5))}
                            className="mx-2 my-2 rounded-sm bg-neutral-800 p-1 text-xs text-neutral-500 opacity-80 hover:text-neutral-400"
                        >
                            random
                        </button>
                    </div>
                    {!invalid && (
                        <button
                            className="my-2 rounded-sm bg-cyan-900 px-4 py-2 text-white hover:bg-cyan-800"
                            onClick={handleSubmit}
                        >
                            join
                        </button>
                    )}
                    {invalid && (
                        <div className="my-2 rounded-sm bg-cyan-800 px-4 py-2 text-gray-200 opacity-40">
                            join
                        </div>
                    )}
                </div>
            </div>
            <Footer />
        </div>
    );
}
