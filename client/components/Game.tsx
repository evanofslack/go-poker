import Chat from "./Chat";
import { useSocket } from "../hooks/useSocket";

export default function Game() {
    const socket = useSocket();

    function handleClick() {
        socket?.send(
            JSON.stringify({
                action: "take-seat",
                params: {
                    buyIn: 1000,
                },
            })
        );
    }

    return (
        <div className="flex h-screen flex-row items-center justify-center">
            <div className="m-24 flex h-1/2 w-3/5 items-center justify-center rounded-3xl bg-green-600">
                <div>
                    Table
                    <button onClick={handleClick}>Sit Down</button>
                </div>
            </div>
            <div className="absolute left-0 bottom-0">
                <Chat />
            </div>
        </div>
    );
}
