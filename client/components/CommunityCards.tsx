import { useContext, useState, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import Card from "./Card";

export default function CommunityCards() {
    const { appState, dispatch } = useContext(AppContext);

    if (appState.game?.communityCards) {
        return (
            <div className="mb-4 flex flex-row p-2 text-2xl text-black">
                {appState.game?.communityCards.map((c, i) => (
                    <div key={i} className="m-0.5">
                        <Card card={c} width={100} height={130} />
                    </div>
                ))}
            </div>
        );
    }
    return null;
}
