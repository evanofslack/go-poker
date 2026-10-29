import React, { useCallback, useContext, useEffect, useState } from "react";
import { useRouter } from "next/router";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Footer from "./Footer";
import { joinTable } from "../actions/actions";
import { Button, PasswordInput, TextInput } from "@mantine/core";
import { invitedPassword, invitedTable } from "../lib/invite";
import CreateTable from "./CreateTable";
import PublicTables from "./PublicTables";
import Features from "./Features";
import { TableSummary } from "../interfaces";

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

    // an invite link fills in the table and any password, so only a name is needed. Wait for
    // the router, whose first replace of the URL would put the password back.
    const router = useRouter();
    useEffect(() => {
        if (!router.isReady) return;
        setTablename(invitedTable());
        // the first call takes the password out of the address bar, so ignore later empty ones
        const invited = invitedPassword();
        if (invited) {
            setPassword(invited);
            setInvitedWithPassword(true);
        }
    }, [router.isReady]);

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

    // picking a listed table joins it once there's a name, unless it needs a password
    const handlePick = (t: TableSummary) => {
        setTablename(t.name);
        dispatch({ type: "setError", payload: null });
        dispatch({ type: "setNeedPassword", payload: t.hasPassword });
        if (socket && connected && username != "" && !t.hasPassword) {
            joinTable(socket, t.name, username);
        }
    };

    const handleUsername = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setUsername(e.target.value);
    }, []);

    const handleTablename = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setTablename(e.target.value);
    }, []);

    return (
        <div className="flex flex-col">
            {/* fills the first screen, so the features start below the fold */}
            <div className="flex min-h-svh flex-col items-center justify-center">
                <h1 className="mb-10 text-4xl font-semibold text-fg sm:mb-16 sm:text-5xl">
                    GoPoker
                </h1>
                <form
                    className="flex w-full max-w-xs flex-col gap-2 px-4 sm:w-auto sm:max-w-none sm:flex-row sm:gap-0 sm:px-0"
                    onSubmit={handleSubmit}
                >
                    <TextInput
                        autoFocus
                        size="md"
                        className="sm:m-2"
                        id="username"
                        maxLength={20}
                        value={username}
                        placeholder="name"
                        onChange={handleUsername}
                    />
                    <TextInput
                        size="md"
                        className="sm:my-2 sm:mr-2 sm:w-48"
                        id="tablename"
                        maxLength={32}
                        value={tablename}
                        placeholder="table"
                        onChange={handleTablename}
                        rightSectionWidth="auto"
                        rightSection={
                            <Button
                                variant="subtle"
                                size="xs"
                                className="mr-1.5"
                                onClick={() => setTablename(randomTableGen(5))}
                            >
                                random
                            </Button>
                        }
                    />
                    {showPassword && (
                        <PasswordInput
                            autoFocus={appState.needPassword}
                            size="md"
                            className="sm:my-2 sm:mr-2 sm:w-40"
                            id="password"
                            maxLength={32}
                            value={password}
                            placeholder="password"
                            aria-label="table password"
                            onChange={(e) => setPassword(e.target.value)}
                        />
                    )}
                    <Button
                        type="button"
                        size="md"
                        className="sm:my-2 sm:mr-2"
                        disabled={invalid}
                        onClick={handleCreate}
                    >
                        create
                    </Button>
                    <Button type="submit" variant="filled" size="md" className="sm:my-2" disabled={invalid}>
                        join
                    </Button>
                </form>
                {appState.error && (
                    <p className="mt-2 px-4 text-center text-error">{appState.error}</p>
                )}
                {!connected && <p className="mt-2 text-subtle">connecting to server…</p>}
                <PublicTables onPick={handlePick} />
                {creating && (
                    <CreateTable
                        tablename={tablename}
                        username={username}
                        onClose={() => setCreating(false)}
                    />
                )}
            </div>
            <Features />
            <Footer />
        </div>
    );
}
