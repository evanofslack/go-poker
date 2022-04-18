import { useState } from "react";
import ChatLog from "./ChatLog";
import Reset from "./Reset";
import Start from "./Start";
import Input from "./Input";
import Table from "./Table";
import { Player } from "../interfaces";

export default function Game() {
    const initialPlayers: (Player | null)[] = [null, null, null, null, null, null];
    const [players, setPlayers] = useState(initialPlayers);

    return (
        <div>
            <Table players={players} setPlayers={setPlayers} />
            <div className="absolute left-0 bottom-0">
                <ChatLog />
            </div>
            <div className="absolute bottom-0 right-0">
                <Input />
            </div>
            <div className="absolute left-0 top-0">
                <Reset />
            </div>
            <div className="absolute bottom-0 right-0">
                <Start players={players} />
            </div>
        </div>
    );
}
