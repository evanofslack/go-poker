import React, { useContext, useState, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import TurnAlert from "./TurnAlert";
import InviteButton from "./InviteButton";
import classNames from "classnames";

export default function GameInfo() {
    const { appState, dispatch } = useContext(AppContext);

    return (
        <div className="invisible p-4 text-right text-subtle sm:visible">
            {appState.game && (
                <p>
                    {appState.game.config.sb}/{appState.game.config.bb} nl texas holdem
                </p>
            )}
            <p className="flex items-center justify-end gap-2">
                <span
                    className={classNames(
                        "h-2 w-2 rounded-full",
                        appState.connection === "open" ? "bg-accent" : "animate-pulse bg-amber-400"
                    )}
                    title={appState.connection === "open" ? "connected" : "reconnecting"}
                />
                table: {appState.table}
            </p>
            <div className="mt-1 flex items-center justify-end gap-2">
                <InviteButton compact />
                <TurnAlert />
            </div>
        </div>
    );
}
