import { useContext } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import TurnAlert from "./TurnAlert";
import InviteButton from "./InviteButton";

export default function GameInfo() {
    const { appState } = useContext(AppContext);
    const config = appState.game?.config;
    const open = appState.connection === "open";

    return (
        <div className="ml-auto min-w-0 text-right text-xs text-subtle sm:text-sm">
            <p className="flex items-center justify-end gap-2">
                <span
                    className={classNames(
                        "h-2 w-2 shrink-0 rounded-full",
                        open ? "bg-accent" : "animate-pulse bg-amber-400"
                    )}
                    title={open ? "connected" : "reconnecting"}
                />
                <span className="truncate">
                    {config && `${config.sb}/${config.bb}`}
                    <span className="hidden sm:inline"> nl texas holdem</span> · {appState.table}
                </span>
            </p>
            <div className="flex items-center justify-end gap-1">
                <InviteButton compact />
                <TurnAlert />
            </div>
        </div>
    );
}
