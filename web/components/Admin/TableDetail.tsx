import { Table } from "@mantine/core";
import type { AdminTable } from "../../interfaces/server";
import { useAdmin } from "../../hooks/useAdmin";
import { AuditFilter, pollMs } from "../../lib/admin";
import { stamp } from "../../lib/time";
import Section from "../Section";
import AuditList from "./AuditList";
import Ledger from "./Ledger";
import LogList from "./LogList";
import { Link, th } from "./Link";

type tableDetailProps = {
    name: string;
    onLookup: (filter: AuditFilter) => void;
};

// TableDetail is one table: who is behind each name, the ledger, the log, and
// its audit. A closed table has only its audit.
export default function TableDetail({ name, onLookup }: tableDetailProps) {
    const { data: t, error } = useAdmin<AdminTable>(`tables/${encodeURIComponent(name)}`, pollMs);

    return (
        <div className="flex flex-col gap-5">
            {error && <p className="text-sm text-subtle">not open now: it may be under closed tables</p>}
            {t && (
                <>
                    <p className="text-xs text-muted">
                        host <span className="text-fg">{t.host || "none"}</span>
                        {t.banker && (
                            <>
                                {" "}
                                · banker <span className="text-fg">{t.banker}</span>
                            </>
                        )}{" "}
                        · {t.settings.sb}/{t.settings.bb} · created from{" "}
                        <Link onClick={() => onLookup({ ip: t.creatorIp })}>{t.creatorIp}</Link> · id {t.id}
                    </p>
                    <Section title="people">
                        <div className="overflow-x-auto rounded-md bg-panel px-1 py-0.5">
                            <Table fz="xs" verticalSpacing={2} horizontalSpacing={6} withRowBorders={false} aria-label="people">
                                <Table.Thead>
                                    <Table.Tr>
                                        <Table.Th className={th}>name</Table.Th>
                                        <Table.Th className={th}>seat</Table.Th>
                                        <Table.Th className={th}>address</Table.Th>
                                        <Table.Th className={th}>connection</Table.Th>
                                        <Table.Th className={th}>since</Table.Th>
                                    </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody className="text-fg">
                                    {t.people.map((p) => (
                                        <Table.Tr key={p.id}>
                                            <Table.Td>
                                                <Link onClick={() => onLookup({ person: p.id })}>{p.name}</Link>
                                            </Table.Td>
                                            <Table.Td>{p.seat || <span className="text-subtle">watching</span>}</Table.Td>
                                            <Table.Td>
                                                {p.connected ? (
                                                    <Link onClick={() => onLookup({ ip: p.ip })}>{p.ip}</Link>
                                                ) : (
                                                    <span className="text-warn">disconnected</span>
                                                )}
                                            </Table.Td>
                                            <Table.Td className="text-muted">{p.conn}</Table.Td>
                                            <Table.Td className="tabular-nums text-muted">{p.since > 0 && stamp(p.since)}</Table.Td>
                                        </Table.Tr>
                                    ))}
                                </Table.Tbody>
                            </Table>
                        </div>
                    </Section>
                    {t.view.ledger.rows.length > 0 && (
                        <Section title="ledger">
                            <Ledger rows={t.view.ledger.rows} />
                        </Section>
                    )}
                    <Section title="log">
                        <LogList logs={t.logs} />
                    </Section>
                </>
            )}
            <Section title="audit">
                <AuditList filter={{ table: name }} onLookup={onLookup} />
            </Section>
        </div>
    );
}
