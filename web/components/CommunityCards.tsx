import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { winningCards } from "../lib/hands";
import Card from "./Card";

export default function CommunityCards() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!view) return null;

    const winning = winningCards(view);
    return (
        <div className="flex flex-row gap-0.5 text-black sm:gap-1" data-board>
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
                    ) : (
                        <Card card={0} placeholder={true} folded={false} hidden={false} />
                    )}
                </div>
            ))}
        </div>
    );
}
