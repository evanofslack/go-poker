import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { winningCards } from "../lib/hands";
import Card from "./Card";

export default function CommunityCards() {
    const { appState, dispatch } = useContext(AppContext);

    const cards = [0, 1, 2, 3, 4];

    if (appState.game?.communityCards) {
        const winning = winningCards(appState.game);
        return (
            <div className="flex flex-row gap-0.5 text-black sm:gap-1">
                {cards.map((num, i) => (
                    <div key={i}>
                        {appState.game?.communityCards[num] ? (
                            // show dealt card
                            <Card
                                card={appState.game.communityCards[num]}
                                placeholder={false}
                                folded={false}
                                hidden={false}
                                highlight={winning.has(String(appState.game.communityCards[num]))}
                            />
                        ) : (
                            // show placeholder
                            <Card
                                card="placeholder"
                                placeholder={true}
                                folded={false}
                                hidden={false}
                            />
                        )}
                    </div>
                ))}
            </div>
        );
    }
    return null;
}
