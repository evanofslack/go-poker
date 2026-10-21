import ChatLog from "./ChatLog";
import Reset from "./Reset";
import GameInfo from "./GameInfo";
import Start from "./Start";
import Input from "./Input";
import Table from "./Table";

export default function Game() {
    return (
        <div className="flex h-dvh flex-col overflow-hidden">
            <header className="flex items-start justify-between gap-2 px-2 pt-1 sm:px-4 sm:pt-2">
                <Reset />
                <GameInfo />
            </header>
            <div className="relative min-h-0 flex-1">
                <Table />
            </div>
            <footer className="relative flex min-h-14 items-end gap-2 p-2 sm:p-4">
                <ChatLog />
                <div className="flex min-w-0 flex-1 justify-end">
                    <Input />
                    <Start />
                </div>
            </footer>
        </div>
    );
}
