import React, { useCallback, useContext, useEffect, useState } from "react";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { joinTable } from "../actions/actions";
import Button from "./Button";
import { invitedPassword, invitedTable } from "../lib/invite";
import CreateTable from "./CreateTable";

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
    const [password, setPassword] = useState("");
    const [invitedWithPassword, setInvitedWithPassword] = useState(false);
    const [creating, setCreating] = useState(false);
    const connected = appState.connection === "open";
    let invalid = username == "" || tablename == "" || !connected;
    const showPassword = appState.needPassword || invitedWithPassword;

    // an invite link fills in the table and any password, so only a name is needed
    useEffect(() => {
        setTablename(invitedTable());
        // the first call takes the password out of the address bar, so ignore later empty ones
        const invited = invitedPassword();
        if (invited) {
            setPassword(invited);
            setInvitedWithPassword(true);
        }
    }, []);

    // the table shows once the server confirms with joined-table
    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (socket && !invalid) {
            joinTable(socket, tablename, username, undefined, password || undefined);
        }
    };

    // creating asks for the table's settings first
    const handleCreate = () => {
        if (invalid) return;
        dispatch({ type: "setError", payload: null });
        setCreating(true);
    };

    const handleUsername = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setUsername(e.target.value);
    }, []);

    const handleTablename = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setTablename(e.target.value);
    }, []);

    return (
        <div className="flex min-h-screen flex-col">
            <div className="flex grow flex-col items-center justify-center ">
                <h1 className="mb-10 text-4xl font-semibold text-white sm:mb-16 sm:text-5xl">
                    GoPoker
                </h1>
                <form
                    className="flex w-full max-w-xs flex-col gap-2 px-4 sm:w-auto sm:max-w-none sm:flex-row sm:gap-0 sm:px-0"
                    onSubmit={handleSubmit}
                >
                    <input
                        autoFocus
                        className="rounded-md bg-raised py-2 pl-4 text-white focus:outline-hidden sm:m-2"
                        id="username"
                        type="text"
                        maxLength={20}
                        value={username}
                        placeholder="name"
                        onChange={handleUsername}
                    />
                    <div className="flex flex-row rounded-md bg-raised text-white sm:my-2 sm:mr-2">
                        <input
                            className="w-full min-w-0 rounded-md bg-raised py-2 pl-4 text-white focus:outline-hidden sm:w-28"
                            id="tablename"
                            type="text"
                            maxLength={32}
                            value={tablename}
                            placeholder="table"
                            onChange={handleTablename}
                        />
                        <Button
                            type="button"
                            variant="ghost"
                            className="mx-2 my-2 bg-surface px-1 py-1 text-xs"
                            onClick={() => setTablename(randomTableGen(5))}
                        >
                            random
                        </Button>
                    </div>
                    {showPassword && (
                        <input
                            autoFocus={appState.needPassword}
                            className="rounded-md bg-raised py-2 pl-4 text-white focus:outline-hidden sm:my-2 sm:mr-2 sm:w-32"
                            id="password"
                            type="password"
                            maxLength={32}
                            value={password}
                            placeholder="password"
                            aria-label="table password"
                            onChange={(e) => setPassword(e.target.value)}
                        />
                    )}
                    <Button
                        type="button"
                        variant="primary"
                        className="sm:my-2 sm:mr-2"
                        disabled={invalid}
                        onClick={handleCreate}
                    >
                        create
                    </Button>
                    <Button type="submit" variant="primary" className="sm:my-2" disabled={invalid}>
                        join
                    </Button>
                </form>
                {appState.error && (
                    <p className="mt-2 px-4 text-center text-rose-400">{appState.error}</p>
                )}
                {!connected && <p className="mt-2 text-subtle">connecting to server…</p>}
                {creating && (
                    <CreateTable
                        tablename={tablename}
                        username={username}
                        onClose={() => setCreating(false)}
                    />
                )}
            </div>
            <Footer />
        </div>
    );
}
