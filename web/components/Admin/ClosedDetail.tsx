import type { AdminClosed, ClosedTable } from "../../interfaces/server";
import { useAdmin } from "../../hooks/useAdmin";
import { AuditFilter } from "../../lib/admin";
import { stamp } from "../../lib/time";
import Section from "../Section";
import AuditList from "./AuditList";
import Ledger from "./Ledger";
import LogList from "./LogList";

type closedDetailProps = {
    table: ClosedTable;
    onLookup: (filter: AuditFilter) => void;
};

// ClosedDetail is what's kept of a closed table: its final ledger, the end of
// its log, and its audit
export default function ClosedDetail({ table, onLookup }: closedDetailProps) {
    const { data: c, error } = useAdmin<AdminClosed>(`closed/${encodeURIComponent(table.id)}`);

    return (
        <div className="flex flex-col gap-5">
            <p className="text-xs text-muted">
                {c && `${c.settings.sb}/${c.settings.bb} · `}
                {stamp(table.created)} to {stamp(table.closed)} · closed {table.reason} ·{" "}
                {table.hands === 1 ? "1 hand" : `${table.hands} hands`} · id {table.id}
            </p>
            {error && <p className="text-sm text-error">{error}</p>}
            {c && (
                <>
                    <Section title="final ledger">
                        {c.ledger ? (
                            <Ledger rows={c.ledger.rows} />
                        ) : (
                            <p className="text-sm text-subtle">saved by another version of the server</p>
                        )}
                    </Section>
                    <Section title="log">
                        <LogList logs={c.logs} />
                    </Section>
                </>
            )}
            <Section title="audit">
                <AuditList filter={{ table: table.id }} onLookup={onLookup} />
            </Section>
        </div>
    );
}
