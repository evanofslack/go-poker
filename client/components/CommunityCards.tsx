import { useContext, useState, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import Card from "./Card";

export default function CommunityCards() {
    const { appState, dispatch } = useContext(AppContext);

    const cards = [0, 1, 2, 3, 4];

    if (appState.game?.communityCards) {
        return (
            <div className="mb-4 flex flex-row p-2 text-2xl text-black">
                {cards.map((num, i) => (
                    <div key={i} className="m-0.5">
                        {appState.game?.communityCards[num] ? (
                            <Card card={appState.game.communityCards[num]} placeholder={false} />
                        ) : (
                            <Card card="placeholder" placeholder={true} />
                        )}
                    </div>
                ))}
            </div>
        );
    }
    return null;
}
