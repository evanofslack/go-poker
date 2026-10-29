import type { Log } from "../../interfaces/table";
import { stamp } from "../../lib/time";

// LogList is a table's log lines, oldest first
export default function LogList({ logs }: { logs: Log[] }) {
    if (logs.length === 0) return <p className="text-sm text-subtle">nothing logged yet</p>;
    return (
        <ol className="flex max-h-48 flex-col gap-0.5 overflow-y-auto rounded-md bg-panel p-2 text-xs">
            {logs.map((l) => (
                <li key={l.id}>
                    <span className="tabular-nums text-subtle">{stamp(l.time)}</span>{" "}
                    <span className="text-fg">{l.message}</span>
                </li>
            ))}
        </ol>
    );
}
