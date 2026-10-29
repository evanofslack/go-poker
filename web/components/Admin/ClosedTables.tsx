import { useState } from "react";
import { Button, Table } from "@mantine/core";
import classNames from "classnames";
import type { ClosedTable } from "../../interfaces/server";
import { useAdmin } from "../../hooks/useAdmin";
import { adminGet, pollMs } from "../../lib/admin";
import { stamp } from "../../lib/time";
import { th } from "./Link";

// pageSize is how many closed tables each load lists, the server's most
const pageSize = 50;

type closedTablesProps = {
    selected: string | null; // a closed table's id
    onSelect: (t: ClosedTable) => void;
};

// ClosedTables lists the tables that closed and are still kept, last closed
// first, with older ones on request. A row opens the table.
export default function ClosedTables({ selected, onSelect }: closedTablesProps) {
    const { data: first, error } = useAdmin<ClosedTable[]>(`closed?limit=${pageSize}`, pollMs);
    const [older, setOlder] = useState<ClosedTable[]>([]);
    const [more, setMore] = useState(true);
    if (error) return <p className="text-sm text-error">{error}</p>;
    if (!first) return <p className="text-sm text-subtle">loading</p>;
    if (first.length === 0) return <p className="text-sm text-subtle">no closed tables kept</p>;

    // a reload of the first page can overlap the older ones
    const tables = [...first, ...older.filter((o) => !first.some((t) => t.id === o.id))];
    const loadOlder = () =>
        adminGet<ClosedTable[]>(`closed?limit=${pageSize}&before=${tables[tables.length - 1].closed}`).then((page) => {
            setOlder((prev) => [...prev, ...page]);
            setMore(page.length === pageSize);
        });

    return (
        <div className="flex flex-col gap-2">
            <div className="overflow-x-auto rounded-md bg-panel px-1 py-0.5">
                <Table fz="xs" verticalSpacing={2} horizontalSpacing={6} withRowBorders={false} highlightOnHover aria-label="closed tables">
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th className={th}>table</Table.Th>
                            <Table.Th className={th}>opened</Table.Th>
                            <Table.Th className={th}>closed</Table.Th>
                            <Table.Th className={th}>why</Table.Th>
                            <Table.Th className={classNames(th, "text-right")}>hands</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody className="text-fg">
                        {tables.map((t) => (
                            <Table.Tr
                                key={t.id}
                                className={classNames("cursor-pointer", t.id === selected && "bg-raised")}
                                onClick={() => onSelect(t)}
                            >
                                <Table.Td className="font-semibold">{t.name}</Table.Td>
                                <Table.Td className="tabular-nums text-muted">{stamp(t.created)}</Table.Td>
                                <Table.Td className="tabular-nums text-muted">{stamp(t.closed)}</Table.Td>
                                <Table.Td>{t.reason}</Table.Td>
                                <Table.Td className="text-right tabular-nums">{t.hands}</Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            </div>
            {more && first.length === pageSize && (
                <Button size="xs" className="self-start" onClick={loadOlder}>
                    older
                </Button>
            )}
        </div>
    );
}
