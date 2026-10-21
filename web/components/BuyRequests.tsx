import { useContext } from "react";
import { hostAction } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Button from "./Button";

// BuyRequests shows the host the rebuys waiting for their ok.
export default function BuyRequests() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!socket || !game || game.host !== appState.username || game.buyRequests.length === 0) {
        return null;
    }
    return (
        <ul className="pointer-events-auto flex flex-col gap-1" aria-label="buy-in requests">
            {game.buyRequests.map((r) => (
                <li
                    key={r.name}
                    className="flex flex-wrap items-center justify-center gap-2 rounded-xl bg-panel/90 px-3 py-1.5 text-xs text-fg sm:text-sm"
                >
                    <span>
                        <span className="font-semibold">{r.name}</span>{" "}
                        {r.seat ? `wants to sit back down with ${r.amount}` : `wants to rebuy for ${r.amount}`}
                    </span>
                    <Button
                        variant="primary"
                        className="px-2 py-0.5 text-xs"
                        onClick={() => hostAction(socket, "approve-buy-in", r.name)}
                    >
                        approve
                    </Button>
                    <Button
                        variant="ghost"
                        className="px-2 py-0.5 text-xs"
                        onClick={() => hostAction(socket, "deny-buy-in", r.name)}
                    >
                        deny
                    </Button>
                </li>
            ))}
        </ul>
    );
}
