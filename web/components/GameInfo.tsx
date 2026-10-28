import { useContext } from "react";
import { FiCopy } from "react-icons/fi";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { copyInvite } from "../lib/invite";
import TurnAlert from "./TurnAlert";
import ColorSchemeToggle from "./ColorSchemeToggle";
import { ActionIcon } from "@mantine/core";

// GameInfo shows the stakes, table and watchers, and below them the connection
// and the header's buttons. A phone has room for the stakes only; the panel
// shows the table's name.
export default function GameInfo() {
    const { appState } = useContext(AppContext);
    const config = appState.view?.settings;
    const watching = appState.view?.watchers.length ?? 0;
    const open = appState.connection === "open";
    const table = appState.table;

    return (
        <div className="ml-auto min-w-0 text-right text-xs text-subtle sm:text-sm">
            {config && (
                <div className="truncate">
                    stakes{" "}
                    <span className="font-semibold text-fg">
                        {config.sb}/{config.bb}
                    </span>
                </div>
            )}
            <div className="truncate phone:hidden">
                table <span className="font-semibold text-fg">{table}</span>
            </div>
            {watching > 0 && <div className="phone:hidden">watching {watching}</div>}
            <div className="flex items-center justify-end gap-1">
                <span className="flex h-[22px] w-[22px] shrink-0 items-center justify-center">
                    <span
                        className={classNames("h-2 w-2 rounded-full", open ? "bg-online" : "animate-pulse bg-offline")}
                        role="status"
                        aria-label={open ? "connected" : "reconnecting"}
                        title={open ? "connected" : "reconnecting"}
                    />
                </span>
                <TurnAlert />
                <ColorSchemeToggle />
                {table && (
                    <ActionIcon
                        size="sm"
                        title="copy invite link"
                        aria-label="copy invite link"
                        onClick={() => copyInvite(table, appState.password)}
                    >
                        <FiCopy />
                    </ActionIcon>
                )}
            </div>
        </div>
    );
}
