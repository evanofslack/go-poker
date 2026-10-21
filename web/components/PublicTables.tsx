import { useContext, useEffect } from "react";
import { FiLock } from "react-icons/fi";
import { listTables } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { TableSummary } from "../interfaces";
import { AppContext } from "../providers/AppStore";

// how often the list refreshes while the home page is open
const pollMs = 5000;

type publicTablesProps = {
    onPick: (table: TableSummary) => void;
};

// column widths, shared by the header and each row
const stakesCol = "w-16 shrink-0 text-right";
const seatsCol = "w-10 shrink-0 text-right";
const statusCol = "w-14 shrink-0 text-right";

// chips shortens big blinds so custom stakes fit their column: 10000 is 10K
const chips = (n: number) => new Intl.NumberFormat("en", { notation: "compact" }).format(n);

export const isFull =(t: TableSummary) => t.players >= t.seats;

function status(t: TableSummary): string {
    if (isFull(t)) return "full";
    if (t.paused) return "paused";
    if (t.running) return "playing";
    return "waiting";
}

// PublicTables lists the public tables under the join form, the server's order
export default function PublicTables({ onPick }: publicTablesProps) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const open = appState.connection === "open";

    // a reconnect changes the socket, which asks again
    useEffect(() => {
        if (!socket || !open) return;
        listTables(socket);
        const timer = setInterval(() => listTables(socket), pollMs);
        return () => clearInterval(timer);
    }, [socket, open]);

    return (
        <section className="mt-10 w-full max-w-md px-4" aria-label="public tables">
            {/* labels the columns below, so it shares their widths */}
            <div className="mb-2 flex items-center gap-3 px-3 text-sm text-muted">
                <h2 className="grow">public tables</h2>
                {appState.tables.length > 0 && (
                    <>
                        <span className={`${stakesCol} text-xs`}>blinds</span>
                        <span className={`${seatsCol} text-xs`}>seats</span>
                        <span className={statusCol} />
                    </>
                )}
            </div>
            {appState.tables.length === 0 ? (
                <p className="text-sm text-subtle">no public tables yet, create one</p>
            ) : (
                <ul className="flex flex-col gap-1">
                    {appState.tables.map((t) => (
                        <li key={t.name}>
                            <button
                                type="button"
                                data-table={t.name}
                                className="flex w-full items-center gap-3 rounded-md bg-raised px-3 py-2 text-left text-sm text-fg hover:outline hover:outline-1 hover:outline-accent"
                                onClick={() => onPick(t)}
                            >
                                <span className="flex min-w-0 grow items-center gap-1">
                                    <span className="truncate font-semibold">{t.name}</span>
                                    {t.hasPassword && (
                                        <FiLock
                                            className="shrink-0 text-muted"
                                            title="needs a password"
                                            aria-label="needs a password"
                                        />
                                    )}
                                </span>
                                <span className={`${stakesCol} text-muted`}>
                                    {chips(t.sb)}/{chips(t.bb)}
                                </span>
                                <span className={seatsCol}>
                                    {t.players}/{t.seats}
                                </span>
                                <span className={`${statusCol} text-xs text-subtle`}>{status(t)}</span>
                            </button>
                        </li>
                    ))}
                </ul>
            )}
        </section>
    );
}
