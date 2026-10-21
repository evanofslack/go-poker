import { useState } from "react";
import ChatLog from "./ChatLog";
import Home from "./Home";
import HostMenu from "./HostMenu";
import SessionEnded from "./SessionEnded";
import GameInfo from "./GameInfo";
import Start from "./Start";
import Input from "./Input";
import Table from "./Table";
import TablePanel from "./TablePanel";

export default function Game() {
    const [tableOpen, setTableOpen] = useState(false);
    return (
        <div className="flex h-dvh flex-col overflow-hidden">
            <header className="flex items-start justify-between gap-2 px-2 pt-1 sm:px-4 sm:pt-2">
                <div className="flex items-center gap-1 sm:gap-2">
                    <Home />
                    <button
                        className="rounded-md px-2 py-1 text-sm text-subtle hover:text-fg"
                        aria-expanded={tableOpen}
                        onClick={() => setTableOpen(!tableOpen)}
                    >
                        table
                    </button>
                    <HostMenu />
                </div>
                <GameInfo />
            </header>
            <div className="relative min-h-0 flex-1">
                <Table />
                {tableOpen && <TablePanel onClose={() => setTableOpen(false)} />}
            </div>
            <footer className="relative flex min-h-14 items-end gap-2 p-2 sm:p-4">
                <ChatLog />
                <div className="flex min-w-0 flex-1 justify-end">
                    <Input />
                    <Start />
                </div>
            </footer>
            <SessionEnded />
        </div>
    );
}
