import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { winningCards } from "../lib/hands";
import Card from "./Card";

export default function CommunityCards() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!view) return null;

    const winning = winningCards(view);
    // a duck hunt fills the slots the hand didn't reach, faded: they count for nothing
    const hunted = (i: number) => view.hunted[i - view.board.length];
    return (
        <div className="relative flex flex-row gap-0.5 text-black sm:gap-1" data-board>
            {[0, 1, 2, 3, 4].map((i) => (
                <div key={i}>
                    {view.board[i] ? (
                        // show dealt card
                        <Card
                            card={view.board[i]}
                            placeholder={false}
                            folded={false}
                            hidden={false}
                            highlight={winning.has(view.board[i])}
                        />
                    ) : hunted(i) ? (
                        <div className="opacity-50" data-hunted>
                            <Card card={hunted(i)} placeholder={false} folded={false} hidden={false} />
                        </div>
                    ) : (
                        <Card card={0} placeholder={true} folded={false} hidden={false} />
                    )}
                </div>
            ))}
            {view.hunted.length > 0 && (
                <span className="absolute inset-x-0 top-full mt-0.5 text-center text-xs text-muted">duck hunt</span>
            )}
        </div>
    );
}
