import { useContext } from "react";
import { duckHunt, showCards } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { Button } from "@mantine/core";
import { bar } from "../lib/theme";
import { bountyHand } from "../lib/settings";

// AfterHand is what a player dealt into the last hand can do before the next:
// show their cards, and duck hunt the rest of the board if it ended early.
export default function AfterHand() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const view = appState.view;
    const me = view?.players.find((p) => p.seat === view.you);
    if (!socket || !view || !me?.dealt || view.street !== "" || view.hand === 0 || appState.connection !== "open") {
        return null;
    }

    const canShow = !me.shown && me.cards[0] !== 0;
    const canHunt = view.settings.duckHunt && view.board.length < 5 && view.hunted.length === 0;
    if (!canShow && !canHunt) return null;
    return (
        <div role="group" aria-label="after the hand" className="flex gap-2">
            {canShow && (
                <Button size="md" className={bar} onClick={() => showCards(socket)}>
                    {view.showClaims.includes(bountyHand(view.settings))
                        ? `show to claim the ${bountyHand(view.settings)}`
                        : view.showClaims.includes("stand-up")
                          ? "show to sit down"
                          : "show cards"}
                </Button>
            )}
            {canHunt && (
                <Button size="md" className={bar} onClick={() => duckHunt(socket)}>
                    duck hunt
                </Button>
            )}
        </div>
    );
}
