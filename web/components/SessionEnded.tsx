import { useContext, useState } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { copyLedger, settleLine, signed } from "../lib/ledger";
import Button from "./Button";
import Dialog from "./Dialog";

// SessionEnded shows everyone the final settle-up once the session ends. The
// players panel keeps it after this is closed.
export default function SessionEnded() {
    const { appState, dispatch } = useContext(AppContext);
    const [closed, setClosed] = useState(false);
    const game = appState.game;
    if (!game?.ended || closed) return null;
    const { rows, banker, balance } = game.ledger;

    return (
        <Dialog title="the session is over" onClose={() => setClosed(true)} wide>
            <p className="mb-3 text-sm text-muted">
                settle up with the banker, <span className="text-fg">{banker || "nobody"}</span>. amounts are in
                chips.
            </p>
            <ul className="mb-3 flex flex-col gap-1 text-sm" aria-label="final ledger">
                {rows.map((r) => (
                    <li key={r.name} className="flex items-baseline gap-2">
                        <span className="font-semibold">{r.name}</span>
                        <span
                            className={classNames(
                                r.net > 0 && "text-green-400",
                                r.net < 0 && "text-rose-400"
                            )}
                        >
                            {signed(r.net)}
                        </span>
                        <span className="ml-auto text-muted">{settleLine(r.name, r.net, banker)}</span>
                    </li>
                ))}
            </ul>
            {balance !== 0 && <p className="mb-3 text-sm text-rose-400">off by {balance}</p>}
            <div className="flex justify-end gap-2">
                <Button variant="ghost" className="px-2 py-1 text-sm" onClick={() => setClosed(true)}>
                    close
                </Button>
                <Button
                    variant="primary"
                    className="px-3 py-1 text-sm"
                    onClick={() => copyLedger(appState.table ?? "", game, dispatch)}
                >
                    copy ledger
                </Button>
            </div>
        </Dialog>
    );
}
