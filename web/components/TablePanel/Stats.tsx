import { useContext } from "react";
import { Table, Tooltip } from "@mantine/core";
import classNames from "classnames";
import { FiAlertCircle } from "react-icons/fi";
import { signed } from "../../lib/ledger";
import { fewHands, names, playerStats, styleColumns, styleStats } from "../../lib/stats";
import { AppContext } from "../../providers/AppStore";
import NetChart from "../NetChart";
import Section from "../Section";
import StyleHeader from "../StyleHeader";

// th and num style the stats tables' header and number cells
const th = "font-normal text-subtle";
const num = "text-right tabular-nums";

// Stats shows each player's net over the hands played, their totals and how they play.
export default function Stats() {
    const { appState } = useContext(AppContext);
    const hands = appState.hands;
    if (hands.length === 0) {
        return <p className="p-2 text-sm text-subtle">stats start after the first hand</p>;
    }
    const all = names(hands);
    const stats = playerStats(hands, appState.view?.ledger.rows ?? []);
    const style = styleStats(appState.totals, all, appState.view?.settings.bb ?? 0);
    const best = style.filter((s) => s.bestHand);
    // since the first hand kept, breaks included; too noisy in the first minutes
    const hours = (Date.now() - hands[0].started) / 3_600_000;
    const perHour = hours >= minRateHours ? Math.round(hands.length / hours) : 0;

    return (
        <div className="flex h-full flex-col gap-4 overflow-y-auto pr-1">
            <p className="text-xs text-muted">
                <span className="text-fg">{hands.length === 1 ? "1 hand" : `${hands.length} hands`}</span>
                {perHour > 0 && ` · ${perHour}/hr · ${duration(hours)}`}
                {hands[0].hand > 1 && ` · from hand ${hands[0].hand}`}
            </p>
            <Section title="net" className="min-h-48 flex-1">
                <div className="min-h-0 flex-1">
                    <NetChart hands={hands} names={all} />
                </div>
            </Section>
            <Section title="players">
                <div className="rounded-md bg-panel px-1 py-0.5">
                    <Table fz="xs" verticalSpacing={2} horizontalSpacing={4} withRowBorders={false} highlightOnHover aria-label="player stats">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th className={th}>player</Table.Th>
                                <Table.Th className={classNames(th, num)}>hands</Table.Th>
                                <Table.Th className={classNames(th, num)}>won</Table.Th>
                                <Table.Th className={classNames(th, num)}>biggest win</Table.Th>
                                <Table.Th className={classNames(th, num)}>net</Table.Th>
                                <Table.Th className={classNames(th, num)}>stack</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody className="text-fg">
                            {stats.map((s) => (
                                <Table.Tr key={s.name}>
                                    <Table.Td>
                                        <span className="font-semibold">{s.name}</span>
                                        {s.name === appState.username && <span className="text-subtle"> you</span>}
                                    </Table.Td>
                                    <Table.Td className={num}>{s.played}</Table.Td>
                                    <Table.Td className={num}>
                                        {s.won}
                                        {s.played > 0 && (
                                            <span className="text-subtle"> {Math.round((100 * s.won) / s.played)}%</span>
                                        )}
                                    </Table.Td>
                                    <Table.Td className={num}>{s.biggestPot || "–"}</Table.Td>
                                    <Table.Td className={classNames(num, s.net > 0 && "text-gain", s.net < 0 && "text-loss")}>
                                        {signed(s.net)}
                                    </Table.Td>
                                    <Table.Td className={num}>{s.stack ?? <span className="text-subtle">left</span>}</Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </div>
            </Section>
            {style.length > 0 && (
                <Section title="play style">
                    <div className="rounded-md bg-panel px-1 py-0.5">
                        <Table.ScrollContainer minWidth={340} type="native">
                            <Table fz="xs" verticalSpacing={2} horizontalSpacing={4} withRowBorders={false} highlightOnHover aria-label="play style">
                                <Table.Thead>
                                    <Table.Tr>
                                        <Table.Th className={classNames(th, "sticky left-0 bg-panel")}>player</Table.Th>
                                        {styleColumns.map(([label, title]) => (
                                            <Table.Th key={label} className={classNames(th, num)}>
                                                <StyleHeader label={label} title={title} />
                                            </Table.Th>
                                        ))}
                                    </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody>
                                    {style.map((s) => (
                                        <Table.Tr key={s.name} className={s.low ? "text-subtle" : "text-fg"}>
                                            <Table.Td className="sticky left-0 bg-panel">
                                                <span className="flex items-center gap-1 font-semibold">
                                                    {s.name}
                                                    {s.low && <FewHands hands={s.hands} />}
                                                </span>
                                            </Table.Td>
                                            <Table.Td className={num}>{s.vpip}</Table.Td>
                                            <Table.Td className={num}>{s.pfr}</Table.Td>
                                            <Table.Td className={num}>{s.threeBet}</Table.Td>
                                            <Table.Td className={num}>{s.af}</Table.Td>
                                            <Table.Td className={num}>{s.wtsd}</Table.Td>
                                            <Table.Td className={num}>{s.wsd}</Table.Td>
                                            <Table.Td className={num}>{s.bb100}</Table.Td>
                                        </Table.Tr>
                                    ))}
                                </Table.Tbody>
                            </Table>
                        </Table.ScrollContainer>
                    </div>
                    {style.some((s) => s.low) && (
                        <p className="-mt-2 text-xs text-subtle">greyed rows have under {fewHands} hands, so they're rough</p>
                    )}
                </Section>
            )}
            {best.length > 0 && (
                <Section title="best hand shown">
                    <ul className="rounded-md bg-panel px-2 py-1 text-xs" aria-label="best hands shown">
                        {best.map((s) => (
                            <li key={s.name} className="flex gap-2">
                                <span className="font-semibold text-fg">{s.name}</span>{" "}
                                <span className="text-muted">{s.bestHand}</span>
                            </li>
                        ))}
                    </ul>
                </Section>
            )}
        </div>
    );
}

// FewHands marks a play style row with too few hands to go by
function FewHands({ hands }: { hands: number }) {
    const text = `only ${hands === 1 ? "1 hand" : `${hands} hands`}, rough until ${fewHands}`;
    return (
        <Tooltip label={text} fz="xs" withArrow events={{ hover: true, focus: true, touch: true }}>
            <span tabIndex={0} role="img" aria-label="few hands" className="inline-flex">
                <FiAlertCircle className="size-3" />
            </span>
        </Tooltip>
    );
}

// duration is how long the hands kept have taken, as "40m" or "2h 5m"
function duration(hours: number) {
    const m = Math.round(hours * 60);
    return m < 60 ? `${m}m` : `${Math.floor(m / 60)}h ${m % 60}m`;
}

// minRateHours is how long hands are played before the hourly rate shows
const minRateHours = 5 / 60;
