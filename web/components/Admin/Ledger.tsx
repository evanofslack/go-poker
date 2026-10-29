import { Table } from "@mantine/core";
import classNames from "classnames";
import type { LedgerRow } from "../../interfaces/table";
import { signed } from "../../lib/ledger";
import { th } from "./Link";

const num = "text-right tabular-nums";

// Ledger is a table's ledger rows: what each name bought in, cashed out and holds
export default function Ledger({ rows }: { rows: LedgerRow[] }) {
    if (rows.length === 0) return <p className="text-sm text-subtle">nobody bought in</p>;
    return (
        <div className="overflow-x-auto rounded-md bg-panel px-1 py-0.5">
            <Table fz="xs" verticalSpacing={2} horizontalSpacing={6} withRowBorders={false} aria-label="ledger">
                <Table.Thead>
                    <Table.Tr>
                        <Table.Th className={th}>name</Table.Th>
                        <Table.Th className={classNames(th, num)}>bought in</Table.Th>
                        <Table.Th className={classNames(th, num)}>cashed out</Table.Th>
                        <Table.Th className={classNames(th, num)}>stack</Table.Th>
                        <Table.Th className={classNames(th, num)}>net</Table.Th>
                    </Table.Tr>
                </Table.Thead>
                <Table.Tbody className="text-fg">
                    {rows.map((r) => (
                        <Table.Tr key={r.name}>
                            <Table.Td className="font-semibold">{r.name}</Table.Td>
                            <Table.Td className={num}>{r.boughtIn}</Table.Td>
                            <Table.Td className={num}>{r.cashedOut}</Table.Td>
                            <Table.Td className={num}>{r.stack}</Table.Td>
                            <Table.Td className={classNames(num, r.net > 0 && "text-gain", r.net < 0 && "text-loss")}>
                                {signed(r.net)}
                            </Table.Td>
                        </Table.Tr>
                    ))}
                </Table.Tbody>
            </Table>
        </div>
    );
}
