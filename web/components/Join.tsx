import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { createTable, joinTable } from "../actions/actions";

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

    // the table shows once the server confirms with joined-table
    const handleSubmit = (e: any, create = false) => {
        e.preventDefault();
        if (socket && !invalid) {
            (create ? createTable : joinTable)(socket, tablename, username);
        }
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
            <div className="flex grow flex-col items-center justify-center ">
                <h1 className="mb-16 text-5xl font-semibold text-white">GoPoker</h1>
                <div className="flex flex-row">
                    <input
                        autoFocus
                        className="m-2 rounded-xs bg-neutral-700 py-2 pl-4 text-white focus:outline-hidden"
                        id="username"
                        type="text"
                        maxLength={20}
                        value={username}
                        placeholder="name"
                        onChange={handleUsername}
                    />
                    <div className=" my-2 mr-2 flex flex-row rounded-xs bg-neutral-700 text-white">
                        <input
                            autoFocus
                            className="w-28 rounded-xs bg-neutral-700 py-2 pl-4 text-white focus:outline-hidden"
                            id="tablename"
                            type="text"
                            maxLength={32}
                            value={tablename}
                            placeholder="table"
                            onChange={handleTablename}
                        />
                        <button
                            onClick={() => setTablename(randomTableGen(5))}
                            className="mx-2 my-2 rounded-xs bg-neutral-800 p-1 text-xs text-neutral-500 opacity-80 hover:text-neutral-400"
                        >
                            random
                        </button>
                    </div>
                    <button
                        className="my-2 mr-2 rounded-xs bg-cyan-900 px-4 py-2 text-white hover:bg-cyan-800 disabled:opacity-40"
                        disabled={invalid}
                        onClick={(e) => handleSubmit(e, true)}
                    >
                        create
                    </button>
                    <button
                        className="my-2 rounded-xs bg-cyan-900 px-4 py-2 text-white hover:bg-cyan-800 disabled:opacity-40"
                        disabled={invalid}
                        onClick={handleSubmit}
                    >
                        join
                    </button>
                </div>
                {appState.error && <p className="mt-2 text-red-400">{appState.error}</p>}
            </div>
            <Footer />
        </div>
    );
}
