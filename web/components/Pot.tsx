import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { settledPots } from "../lib/hands";

const chip = "flex h-10 items-center justify-center rounded-full bg-felt-dark px-5 text-on-felt short:h-7 short:px-3";

export default function Pot() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!view || !view.running) {
        return null;
    }

    // between hands, the winners' seats say what they won
    if (settledPots(view).length) {
        return null;
    }

    // every chip committed this hand, including the current street
    const total = view.players.reduce((sum, p) => sum + p.committed, 0);
    return <p className={chip + " min-w-24 text-2xl font-semibold short:min-w-16 short:text-base"}>{total}</p>;
}
