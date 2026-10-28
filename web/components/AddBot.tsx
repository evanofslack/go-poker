import { useState } from "react";
import { addBot } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { View } from "../interfaces/index";
import { randomBotName } from "../lib/bots";
import { TextInput } from "@mantine/core";
import BuyIn from "./BuyIn";
import Segmented from "./Segmented";

// maxNameLength mirrors maxUsernameLen on the server
const maxNameLength = 20;

const levels = [
    { label: "easy", value: "easy" },
    { label: "medium", value: "medium" },
];

type addBotProps = {
    seat: number;
    view: View;
    onDone: () => void;
};

// AddBot is the host's form for sitting a bot down: its level, name and buy-in
export default function AddBot({ seat, view: game, onDone }: addBotProps) {
    const socket = useSocket();
    const [name, setName] = useState(() => randomBotName(game.players.map((p) => p.name)));
    const [level, setLevel] = useState("medium");

    return (
        <div className="flex flex-col gap-2">
            <div className="px-3">
                <Segmented
                    legend="level"
                    name="bot-level"
                    options={levels}
                    value={level}
                    onChange={setLevel}
                />
            </div>
            <TextInput
                label="name"
                size="xs"
                className="px-3"
                classNames={{ label: "font-normal text-subtle" }}
                maxLength={maxNameLength}
                value={name}
                onChange={(e) => setName(e.target.value)}
            />
            <BuyIn
                min={game.settings.minBuy}
                max={game.settings.maxBuy}
                initial={100 * game.settings.bb}
                label="add bot"
                onSubmit={(amount) => {
                    if (socket && name.trim()) {
                        addBot(socket, seat, amount, name.trim(), level);
                    }
                    onDone();
                }}
                onCancel={onDone}
            />
        </div>
    );
}
