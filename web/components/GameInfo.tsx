import { useContext } from "react";
import { FiCopy } from "react-icons/fi";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { copyInvite } from "../lib/invite";
import TurnAlert from "./TurnAlert";

// GameInfo shows the stakes, table and connection, and copies the invite link
export default function GameInfo() {
    const { appState, dispatch } = useContext(AppContext);
    const config = appState.view?.settings;
    const watching = appState.view?.watchers.length ?? 0;
    const open = appState.connection === "open";
    const table = appState.table;

    return (
        <div className="ml-auto min-w-0 text-right text-xs text-subtle sm:text-sm">
            <p className="flex items-center justify-end gap-2">
                <span
                    className={classNames(
                        "h-2 w-2 shrink-0 rounded-full",
                        open ? "bg-green-500" : "animate-pulse bg-red-500"
                    )}
                    role="status"
                    aria-label={open ? "connected" : "reconnecting"}
                    title={open ? "connected" : "reconnecting"}
                />
                {config && (
                    <span className="truncate">
                        <span className="hidden sm:inline">stakes </span>
                        {config.sb}/{config.bb}
                    </span>
                )}
            </p>
            <div className="flex items-center justify-end gap-1">
                {/* a phone has no room for the table's name, which the table menu shows */}
                {watching > 0 && (
                    <span className="mr-1 shrink-0 whitespace-nowrap">
                        watching {watching}
                        <span className="phone:hidden"> ·</span>
                    </span>
                )}
                <span className="truncate phone:hidden">
                    table <span className="text-muted">{table}</span>
                </span>
                {table && (
                    <button
                        className="p-1 hover:text-fg"
                        title="copy invite link"
                        aria-label="copy invite link"
                        onClick={() => copyInvite(table, appState.password, dispatch)}
                    >
                        <FiCopy />
                    </button>
                )}
                <TurnAlert />
            </div>
        </div>
    );
}
