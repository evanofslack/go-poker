import { useRef, useState } from "react";
import { Button } from "@mantine/core";
import Layout from "../components/Layout";
import Section from "../components/Section";
import Overview from "../components/Admin/Overview";
import ClosedTables from "../components/Admin/ClosedTables";
import TableDetail from "../components/Admin/TableDetail";
import ClosedDetail from "../components/Admin/ClosedDetail";
import Lookup from "../components/Admin/Lookup";
import type { ClosedTable } from "../interfaces/server";
import { AuditFilter } from "../lib/admin";

// Selected is the table shown in full: an open one by name, or a closed one
type Selected = { open: string } | { closed: ClosedTable };

// AdminPage is for whoever runs the server: the open and closed tables, who is
// behind each name, and the audit. It's only served on ADMIN_ADDR.
export default function AdminPage() {
    const [selected, setSelected] = useState<Selected | null>(null);
    const [lookup, setLookup] = useState<AuditFilter | null>(null);
    const tableRef = useRef<HTMLDivElement>(null);
    const lookupRef = useRef<HTMLDivElement>(null);

    const show = (s: Selected) => {
        setSelected(s);
        requestAnimationFrame(() => tableRef.current?.scrollIntoView({ behavior: "smooth" }));
    };
    const showLookup = (filter: AuditFilter) => {
        setLookup(filter);
        requestAnimationFrame(() => lookupRef.current?.scrollIntoView({ behavior: "smooth" }));
    };
    const open = selected && "open" in selected ? selected.open : null;
    const closed = selected && "closed" in selected ? selected.closed : null;

    return (
        <Layout title="Poker admin">
            <div className="mx-auto flex min-h-screen max-w-6xl flex-col gap-5 bg-surface p-4">
                <h1 className="text-lg font-semibold text-fg">admin</h1>
                <Section title="tables">
                    <Overview selected={open} onSelect={(name) => show({ open: name })} onLookup={showLookup} />
                </Section>
                <Section title="closed tables">
                    <ClosedTables selected={closed?.id ?? null} onSelect={(t) => show({ closed: t })} />
                </Section>
                {selected && (
                    <div ref={tableRef}>
                        <Section title={open ?? `${closed!.name} (closed)`}>
                            {open && <TableDetail key={open} name={open} onLookup={showLookup} />}
                            {closed && <ClosedDetail key={closed.id} table={closed} onLookup={showLookup} />}
                            <Button variant="subtle" size="xs" className="self-start" onClick={() => setSelected(null)}>
                                close
                            </Button>
                        </Section>
                    </div>
                )}
                <div ref={lookupRef}>
                    <Section title="lookup">
                        <Lookup filter={lookup} onSearch={setLookup} onTable={(name) => show({ open: name })} />
                    </Section>
                </div>
            </div>
        </Layout>
    );
}
