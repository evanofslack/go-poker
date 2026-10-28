import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { settledPots } from "../lib/hands";
import Chips from "./Chips";
import { useTween } from "../hooks/useTween";

const chip = "flex h-10 items-center justify-center gap-2 rounded-full bg-felt-dark px-5 text-on-felt short:h-7 short:gap-1.5 short:px-3";

export default function Pot() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    // every chip committed this hand, including the current street, counted up
    // as it grows; a new hand starts from its own blinds
    const committed = view?.players.reduce((sum, p) => sum + p.committed, 0) ?? 0;
    const total = useTween(committed, view?.hand);
    if (!view || !view.running) {
        return null;
    }

    // between hands, the winners' seats say what they won
    if (settledPots(view).length) {
        return null;
    }

    return (
        <p className={chip + " min-w-24 text-2xl font-semibold short:min-w-16 short:text-base"}>
            {committed > 0 && <Chips amount={committed} bb={view.settings.bb} className="size-5 short:size-4" />}
            {total}
        </p>
    );
}
