import React, { useContext, useState, useEffect } from "react";
import { AppContext } from "../providers/AppStore";

export default function GameInfo() {
    const { appState, dispatch } = useContext(AppContext);

    return (
        <div className="invisible p-4 text-right text-zinc-600 sm:visible">
            {appState.game && (
                <p>
                    {appState.game.config.sb}/{appState.game.config.bb} nl texas holdem
                </p>
            )}
            <p>table: {appState.table}</p>
        </div>
    );
}
