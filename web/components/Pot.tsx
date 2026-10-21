import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { settledPots } from "../lib/hands";

const chip = "flex h-10 items-center justify-center rounded-full bg-green-900 px-5 text-white short:h-7 short:px-3";

export default function Pot() {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game || !game.running) {
        return null;
    }

    // between hands, the winners' seats say what they won
    if (settledPots(game).length) {
        return null;
    }

    // every chip committed this hand, including the current street
    const total = game.players.reduce((sum, p) => sum + p.totalBet, 0);
    return <p className={chip + " min-w-24 text-2xl font-semibold short:min-w-16 short:text-base"}>{total}</p>;
}
