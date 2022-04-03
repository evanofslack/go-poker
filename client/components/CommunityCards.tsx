import { useContext, useState, useEffect } from "react";
import { AppContext } from "../providers/AppStore";

export default function CommunityCards() {
    const { appState, dispatch } = useContext(AppContext);

    if (appState.game?.communityCards) {
        return (
            <div className="mt-6 mb-4 flex flex-row p-2 text-2xl text-black">
                {appState.game?.communityCards.map((c, i) => (
                    <div key={i}>
                        {c == "2\u0000" ? (
                            <p className="mx-2 px-2 py-6">{null}</p>
                        ) : (
                            <p className="mx-2 border bg-white px-2 py-6">{c}</p>
                        )}
                    </div>
                ))}
            </div>
        );
    }
    return null;
}
