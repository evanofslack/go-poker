import { Table } from "@mantine/core";
import classNames from "classnames";
import type { AdminOverview, AdminTableRow } from "../../interfaces/server";
import { useAdmin } from "../../hooks/useAdmin";
import { AuditFilter, pollMs } from "../../lib/admin";
import { stamp } from "../../lib/time";
import { Link, th } from "./Link";

const num = "text-right tabular-nums";

type overviewProps = {
    selected: string | null;
    onSelect: (name: string) => void;
    onLookup: (filter: AuditFilter) => void;
};

// Overview is the server and its open tables, reloaded every few seconds. A
// row opens the table.
export default function Overview({ selected, onSelect, onLookup }: overviewProps) {
    const { data, error } = useAdmin<AdminOverview>("overview", pollMs);
    if (error) return <p className="text-sm text-error">{error}</p>;
    if (!data) return <p className="text-sm text-subtle">loading</p>;
    const ips = Object.keys(data.connsByIp).length;

    return (
        <div className="flex flex-col gap-2">
            <p className="text-xs text-muted">
                {data.store} store · up since {stamp(data.started)} · {data.connections}{" "}
                {data.connections === 1 ? "connection" : "connections"} from {ips} {ips === 1 ? "address" : "addresses"}
            </p>
            {data.tables.length === 0 ? (
                <p className="text-sm text-subtle">no open tables</p>
            ) : (
                <div className="overflow-x-auto rounded-md bg-panel px-1 py-0.5">
                    <Table fz="xs" verticalSpacing={2} horizontalSpacing={6} withRowBorders={false} highlightOnHover aria-label="tables">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th className={th}>table</Table.Th>
                                <Table.Th className={th}>stakes</Table.Th>
                                <Table.Th className={classNames(th, num)}>seated</Table.Th>
                                <Table.Th className={classNames(th, num)}>watching</Table.Th>
                                <Table.Th className={classNames(th, num)}>connections</Table.Th>
                                <Table.Th className={th}>status</Table.Th>
                                <Table.Th className={th}>created from</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody className="text-fg">
                            {data.tables.map((t) => (
                                <Table.Tr
                                    key={t.id}
                                    className={classNames("cursor-pointer", t.summary.name === selected && "bg-raised")}
                                    onClick={() => onSelect(t.summary.name)}
                                >
                                    <Table.Td className="font-semibold">
                                        {t.summary.name}
                                        {!t.public && <span className="font-normal text-subtle"> private</span>}
                                        {t.summary.hasPassword && <span className="font-normal text-subtle"> password</span>}
                                    </Table.Td>
                                    <Table.Td className="tabular-nums">
                                        {t.summary.sb}/{t.summary.bb}
                                        {t.summary.ante > 0 && ` ante ${t.summary.ante}`}
                                    </Table.Td>
                                    <Table.Td className={num}>
                                        {t.summary.players}/{t.summary.seats}
                                        {t.summary.bots > 0 && <span className="text-subtle"> {t.summary.bots} bots</span>}
                                    </Table.Td>
                                    <Table.Td className={num}>{t.summary.watchers}</Table.Td>
                                    <Table.Td className={num}>{t.connections}</Table.Td>
                                    <Table.Td>{status(t)}</Table.Td>
                                    <Table.Td>
                                        <Link onClick={() => onLookup({ ip: t.creatorIp })}>{t.creatorIp}</Link>
                                    </Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </div>
            )}
        </div>
    );
}

function status(t: AdminTableRow) {
    if (t.ended) return "ended";
    if (t.summary.paused) return "paused";
    return t.summary.running ? "playing" : "waiting";
}
