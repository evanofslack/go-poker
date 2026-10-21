import { useState } from "react";
import { addBot } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game } from "../interfaces/index";
import { randomBotName } from "../lib/bots";
import BuyIn from "./BuyIn";

// maxNameLength mirrors maxUsernameLen on the server
const maxNameLength = 20;

type addBotProps = {
    seat: number;
    game: Game;
    onDone: () => void;
};

// AddBot is the host's form for sitting a bot down: its name and buy-in
export default function AddBot({ seat, game, onDone }: addBotProps) {
    const socket = useSocket();
    const [name, setName] = useState(() => randomBotName(game.players.map((p) => p.username)));

    return (
        <div className="flex flex-col gap-2">
            <label className="flex flex-col gap-1 px-3 text-sm text-subtle">
                name
                <input
                    className="rounded-md bg-zinc-600 p-1 text-fg focus:outline-hidden"
                    maxLength={maxNameLength}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                />
            </label>
            <BuyIn
                min={game.settings.minBuy}
                max={game.settings.maxBuy}
                initial={100 * game.settings.bb}
                label="add bot"
                onSubmit={(amount) => {
                    if (socket && name.trim()) {
                        addBot(socket, seat, amount, name.trim(), "easy");
                    }
                    onDone();
                }}
                onCancel={onDone}
            />
        </div>
    );
}
