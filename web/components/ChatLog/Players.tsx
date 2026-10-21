import { useContext } from "react";
import classNames from "classnames";
import { AppContext } from "../../providers/AppStore";
import { ledgerText, playerLines, settleLine, signed } from "../../lib/ledger";
import Button from "../Button";

// Players lists everyone at the table with their chips in and out, and how
// each settles up with the banker
export default function Players() {
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;
    if (!game) return null;
    const { banker, balance } = game.ledger;
    const lines = playerLines(game);

    const copy = async () => {
        const text = ledgerText(appState.table ?? "", game);
        try {
            await navigator.clipboard.writeText(text);
            dispatch({ type: "addToast", payload: { message: "ledger copied", kind: "info" } });
        } catch {
            dispatch({ type: "addToast", payload: { message: "couldn't copy the ledger", kind: "error" } });
        }
    };

    return (
        <div className="flex h-full flex-col bg-surface">
            <ul className="min-h-0 flex-1 overflow-auto p-2" aria-label="players">
                {lines.map(({ name, status, seat, row, bot }) => (
                    <li key={name} className="border-b border-raised py-1.5 last:border-b-0">
                        <div className="flex items-baseline gap-2">
                            <span className="truncate font-semibold text-fg">{name}</span>
                            {name === appState.username && <span className="text-xs text-subtle">you</span>}
                            {bot && <span className="text-xs text-subtle">{bot} bot</span>}
                            <span className="text-xs text-subtle">
                                {status}
                                {seat !== null && ` · seat ${seat}`}
                            </span>
                            {row && (
                                <span
                                    className={classNames(
                                        "ml-auto font-semibold",
                                        row.net > 0 && "text-green-400",
                                        row.net < 0 && "text-rose-400"
                                    )}
                                >
                                    {signed(row.net)}
                                </span>
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
                <span className={balance === 0 ? "text-subtle" : "text-rose-400"}>
                    {balance === 0 ? "· balanced" : `· off by ${balance}`}
                </span>
                <Button className="ml-auto px-2 py-0.5 text-xs" onClick={copy}>
                    copy ledger
                </Button>
            </div>
        </div>
    );
}
