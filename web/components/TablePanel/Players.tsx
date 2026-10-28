import { useContext, useState } from "react";
import classNames from "classnames";
import { AppContext } from "../../providers/AppStore";
import { copyLedger, playerLines, settleLine, signed } from "../../lib/ledger";
import { ActionIcon, Button } from "@mantine/core";
import { FiMoreHorizontal } from "react-icons/fi";
import ManagePlayer from "./ManagePlayer";

// Players lists everyone at the table with their chips in and out, and how
// each settles up with the banker
export default function Players() {
    const { appState } = useContext(AppContext);
    const game = appState.view;
    const [managing, setManaging] = useState<string | null>(null);
    if (!game) return null;
    const { banker, balance } = game.ledger;
    const lines = playerLines(game);
    const host = game.host === appState.username;
    const managed = lines.find((l) => l.name === managing);

    return (
        <div className="flex h-full flex-col bg-surface">
            <ul className="min-h-0 flex-1 overflow-auto p-2" aria-label="players">
                {lines.map(({ name, status, seat, row, bot }) => (
                    <li key={name} className="border-b border-raised py-1.5 last:border-b-0">
                        <div className="flex items-baseline gap-2">
                            <span className="truncate font-semibold text-fg">{name}</span>
                            {name === appState.username && <span className="text-xs text-subtle">you</span>}
                            {bot && <span className="text-xs text-subtle">{bot} bot</span>}
                            {name === game.host && <span className="text-xs text-accent">host</span>}
                            <span className="text-xs text-subtle">
                                {status}
                                {seat !== null && ` · seat ${seat}`}
                            </span>
                            {row && (
                                <span
                                    className={classNames(
                                        "ml-auto font-semibold",
                                        row.net > 0 && "text-gain",
                                        row.net < 0 && "text-loss"
                                    )}
                                >
                                    {signed(row.net)}
                                </span>
                            )}
                            {host && (
                                <ActionIcon
                                    size="sm"
                                    className={classNames(!row && "ml-auto")}
                                    title={`manage ${name}`}
                                    aria-label={`manage ${name}`}
                                    onClick={() => setManaging(name)}
                                >
                                    <FiMoreHorizontal />
                                </ActionIcon>
                            )}
                        </div>
                        {row && (
                            <div className="flex gap-2 text-xs text-subtle">
                                <span>
                                    stack {row.stack} · in {row.boughtIn} · out {row.cashedOut}
                                </span>
                                <span className="ml-auto text-muted">{settleLine(name, row.net, banker)}</span>
                            </div>
                        )}
                    </li>
                ))}
            </ul>
            <div className="flex items-center gap-2 border-t border-raised px-2 py-1.5 text-xs">
                <span>
                    banker <span className="text-fg">{banker || "none"}</span>
                </span>
                <span className={balance === 0 ? "text-subtle" : "text-error"}>
                    {balance === 0 ? "· balanced" : `· off by ${balance}`}
                </span>
                <Button size="xs" className="ml-auto" onClick={() => copyLedger(appState.table ?? "", game)}>
                    copy ledger
                </Button>
            </div>
            {managed && <ManagePlayer line={managed} onClose={() => setManaging(null)} />}
        </div>
    );
}
