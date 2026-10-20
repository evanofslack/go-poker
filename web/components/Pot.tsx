import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { potLabel, settledPots } from "../lib/hands";

const chip = "flex h-10 items-center justify-center rounded-full bg-green-900 px-5 text-white";

export default function Pot() {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game || !game.running) {
        return null;
    }

    // between hands, say who won what
    const settled = settledPots(game);
    if (settled.length) {
        return (
            <div className="flex flex-col items-center gap-1">
                {settled.map((pot, i) => (
                    <p key={i} className={chip + " text-lg"}>
                        {potLabel(game, pot, game.pots.indexOf(pot))}
                    </p>
                ))}
            </div>
        );
    }

    // every chip committed this hand, including the current street
    const total = game.players.reduce((sum, p) => sum + p.totalBet, 0);
    return <p className={chip + " min-w-24 text-2xl font-semibold"}>{total}</p>;
}
