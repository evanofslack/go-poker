import { Fragment, useEffect, useState } from "react";
import { Button, Checkbox, Table, TextInput } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import classNames from "classnames";
import type { AuditEntry } from "../../interfaces/server";
import { AuditFilter, adminGet, auditPath } from "../../lib/admin";
import { stamp } from "../../lib/time";
import { Link, th } from "./Link";

// pageSize is how many entries each load adds
const pageSize = 100;

type auditListProps = {
    filter: AuditFilter; // what the list is of: a table, an address, a name
    onTable?: (name: string) => void; // shows the table column, linking to each table
    onLookup: (filter: AuditFilter) => void;
};

// AuditList shows the audit entries filter picks, newest first, narrowed by
// kind, name and refused, with older pages on request. A row opens to show
// its command and what it logged.
export default function AuditList({ filter, onTable, onLookup }: auditListProps) {
    const [kind, setKind] = useState("");
    const [name, setName] = useState("");
    const [refused, setRefused] = useState(false);
    const [debounced] = useDebouncedValue({ kind: kind.trim(), name: name.trim() }, 300);
    const [entries, setEntries] = useState<AuditEntry[]>([]);
    const [more, setMore] = useState(false);
    const [error, setError] = useState("");
    const [open, setOpen] = useState<number | null>(null);
    const [reloads, setReloads] = useState(0);

    const query: AuditFilter = { ...filter, refused, kind: debounced.kind, name: filter.name || debounced.name };
    const key = JSON.stringify(query);

    const load = (before?: number) =>
        adminGet<AuditEntry[]>(auditPath(query, pageSize, before)).then(
            (page) => {
                setEntries((prev) => (before ? [...prev, ...page] : page));
                setMore(page.length === pageSize);
                setError("");
            },
            (e: Error) => setError(e.message)
        );

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [key, reloads]);

    return (
        <div className="flex flex-col gap-2">
            <div className="flex flex-wrap items-end gap-3">
                <TextInput size="xs" label="kind" placeholder="take-seat" value={kind} onChange={(e) => setKind(e.currentTarget.value)} />
                {!filter.name && (
                    <TextInput size="xs" label="name" value={name} onChange={(e) => setName(e.currentTarget.value)} />
                )}
                <Checkbox size="xs" className="pb-1.5" label="refused only" checked={refused} onChange={(e) => setRefused(e.currentTarget.checked)} />
                <Button size="xs" className="ml-auto" onClick={() => setReloads((n) => n + 1)}>
                    refresh
                </Button>
            </div>
            {error && <p className="text-sm text-error">{error}</p>}
            {entries.length === 0 && !error ? (
                <p className="text-sm text-subtle">nothing recorded</p>
            ) : (
                <div className="overflow-x-auto rounded-md bg-panel px-1 py-0.5">
                    <Table fz="xs" verticalSpacing={2} horizontalSpacing={6} withRowBorders={false} highlightOnHover aria-label="audit">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th className={th}>time</Table.Th>
                                {onTable && <Table.Th className={th}>table</Table.Th>}
                                <Table.Th className={th}>name</Table.Th>
                                <Table.Th className={th}>address</Table.Th>
                                <Table.Th className={th}>connection</Table.Th>
                                <Table.Th className={th}>kind</Table.Th>
                                <Table.Th className={th}>refused</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody className="text-fg">
                            {entries.map((e) => (
                                <Fragment key={e.id}>
                                    <Table.Tr className="cursor-pointer" onClick={() => setOpen(open === e.id ? null : e.id)}>
                                        <Table.Td className="whitespace-nowrap tabular-nums text-muted">{stamp(e.time)}</Table.Td>
                                        {onTable && (
                                            <Table.Td>
                                                {e.tableName && <Link onClick={() => onTable(e.tableName)}>{e.tableName}</Link>}
                                            </Table.Td>
                                        )}
                                        <Table.Td>{e.name && <Link onClick={() => onLookup({ name: e.name })}>{e.name}</Link>}</Table.Td>
                                        <Table.Td>{e.ip && <Link onClick={() => onLookup({ ip: e.ip })}>{e.ip}</Link>}</Table.Td>
                                        <Table.Td className="text-muted">{e.conn}</Table.Td>
                                        <Table.Td className="font-semibold">{e.kind}</Table.Td>
                                        <Table.Td className={classNames(e.error && "text-error")}>{e.error}</Table.Td>
                                    </Table.Tr>
                                    {open === e.id && (
                                        <Table.Tr>
                                            <Table.Td colSpan={onTable ? 7 : 6}>
                                                <pre className="overflow-x-auto rounded bg-raised p-2 text-xs text-muted">
                                                    {JSON.stringify({ person: e.person, table: e.table, ...(e.data as object) }, null, 2)}
                                                </pre>
                                            </Table.Td>
                                        </Table.Tr>
                                    )}
                                </Fragment>
                            ))}
                        </Table.Tbody>
                    </Table>
                </div>
            )}
            {more && (
                <Button size="xs" className="self-start" onClick={() => load(entries[entries.length - 1].id)}>
                    older
                </Button>
            )}
        </div>
    );
}
