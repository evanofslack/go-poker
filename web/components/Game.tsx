import { useState } from "react";
import { FiChevronDown } from "react-icons/fi";
import AfterHand from "./AfterHand";
import StraddlePrompt from "./StraddlePrompt";
import RunItPrompt from "./RunItPrompt";
import ChatLog from "./ChatLog";
import Home from "./Home";
import SessionEnded from "./SessionEnded";
import GameInfo from "./GameInfo";
import Start from "./Start";
import StandUpAgain from "./StandUpAgain";
import Input from "./Input";
import Table from "./Table";
import TablePanel from "./TablePanel";
import { Button } from "@mantine/core";

export default function Game() {
    const [tableOpen, setTableOpen] = useState(false);
    return (
        <div className="flex h-dvh flex-col overflow-hidden">
            <header className="flex items-start justify-between gap-2 px-2 pt-1 sm:px-4 sm:pt-2">
                <div className="flex items-center gap-1 sm:gap-2">
                    <Home />
                    <Button
                        variant="subtle"
                        size="compact-sm"
                        rightSection={<FiChevronDown />}
                        aria-expanded={tableOpen}
                        onClick={() => setTableOpen(!tableOpen)}
                    >
                        table
                    </Button>
                </div>
                <GameInfo />
            </header>
            <div className="relative min-h-0 flex-1">
                <Table />
                {tableOpen && <TablePanel onClose={() => setTableOpen(false)} />}
            </div>
            {/* fixed height, so the table doesn't resize as the controls change. On a tall
                screen it's high enough for the raise controls, which float over the table
                elsewhere. */}
            <footer className="relative flex h-16 shrink-0 items-end gap-2 p-2 sm:h-20 sm:p-4 tall:h-28">
                <ChatLog />
                <StandUpAgain />
                <div className="flex min-w-0 flex-1 justify-end gap-2">
                    <Input />
                    <StraddlePrompt />
                    <RunItPrompt />
                    <AfterHand />
                    <Start />
                </div>
            </footer>
            <SessionEnded />
        </div>
    );
}
