import { useRef, useState } from "react";
import { Button } from "@mantine/core";
import Layout from "../components/Layout";
import Section from "../components/Section";
import Overview from "../components/Admin/Overview";
import TableDetail from "../components/Admin/TableDetail";
import Lookup from "../components/Admin/Lookup";
import { AuditFilter } from "../lib/admin";

// AdminPage is for whoever runs the server: the open tables, who is behind
// each name, and the audit. It's only served on ADMIN_ADDR.
export default function AdminPage() {
    const [table, setTable] = useState<string | null>(null);
    const [lookup, setLookup] = useState<AuditFilter | null>(null);
    const tableRef = useRef<HTMLDivElement>(null);
    const lookupRef = useRef<HTMLDivElement>(null);

    const showTable = (name: string) => {
        setTable(name);
        requestAnimationFrame(() => tableRef.current?.scrollIntoView({ behavior: "smooth" }));
    };
    const showLookup = (filter: AuditFilter) => {
        setLookup(filter);
        requestAnimationFrame(() => lookupRef.current?.scrollIntoView({ behavior: "smooth" }));
    };

    return (
        <Layout title="Poker admin">
            <div className="mx-auto flex min-h-screen max-w-6xl flex-col gap-5 bg-surface p-4">
                <h1 className="text-lg font-semibold text-fg">admin</h1>
                <Section title="tables">
                    <Overview selected={table} onSelect={showTable} onLookup={showLookup} />
                </Section>
                {table && (
                    <div ref={tableRef}>
                        <Section title={table}>
                            <TableDetail key={table} name={table} onLookup={showLookup} />
                            <Button variant="subtle" size="xs" className="self-start" onClick={() => setTable(null)}>
                                close
                            </Button>
                        </Section>
                    </div>
                )}
                <div ref={lookupRef}>
                    <Section title="lookup">
                        <Lookup filter={lookup} onSearch={setLookup} onTable={showTable} />
                    </Section>
                </div>
            </div>
        </Layout>
    );
}
