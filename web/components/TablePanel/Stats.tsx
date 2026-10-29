import { useContext } from "react";
import { fewHands, names, playerStats, seriesColor, styleColumns, styleStats } from "../../lib/stats";
import { AppContext } from "../../providers/AppStore";
import StackChart from "../StackChart";

// Stats shows each player's chips over the hands played, and their totals.
export default function Stats() {
    const { appState } = useContext(AppContext);
    const hands = appState.hands;
    if (hands.length === 0) {
        return <p className="p-2 text-sm text-subtle">stats start after the first hand</p>;
    }
    const all = names(hands);
    const stats = playerStats(hands);
    const style = styleStats(appState.totals, all, appState.view?.settings.bb ?? 0);
    const best = style.filter((s) => s.bestHand);
    // since the first hand kept, breaks included; too noisy in the first minutes
    const hours = (Date.now() - hands[0].started) / 3_600_000;
    const perHour = hours >= minRateHours ? Math.round(hands.length / hours) : 0;

    return (
        <div className="flex h-full flex-col gap-2 overflow-y-auto pr-1">
            <ul className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted" aria-label="chart legend">
                {all.map((n, i) => (
                    <li key={n} className="flex items-center gap-1">
                        <span className="inline-block h-0.5 w-3 shrink-0 rounded-full" style={{ background: seriesColor(i) }} />
                        {n}
                    </li>
                ))}
            </ul>
            <StackChart hands={hands} names={all} />
            <table className="w-full text-left text-xs" aria-label="player stats">
                <thead className="text-subtle">
                    <tr>
                        <th className="font-normal">player</th>
                        <th className="text-right font-normal">hands</th>
                        <th className="text-right font-normal">won</th>
                        <th className="text-right font-normal">biggest pot</th>
                        <th className="text-right font-normal">stack</th>
                    </tr>
                </thead>
                <tbody className="text-fg tabular-nums">
                    {stats.map((s) => (
                        <tr key={s.name}>
                            <td className="font-semibold">{s.name}</td>
                            <td className="text-right">{s.played}</td>
                            <td className="text-right">
                                {s.won}
                                {s.played > 0 && (
                                    <span className="text-subtle"> {Math.round((100 * s.won) / s.played)}%</span>
                                )}
                            </td>
                            <td className="text-right">{s.biggestPot || "–"}</td>
                            <td className="text-right">{s.stack ?? <span className="text-subtle">left</span>}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
            {style.length > 0 && (
                <div className="shrink-0 overflow-x-auto">
                    <table className="w-full text-left text-xs" aria-label="play style">
                        <thead className="text-subtle">
                            <tr>
                                <th className="font-normal">play style</th>
                                {styleColumns.map(([label, title]) => (
                                    <th key={label} className="text-right font-normal">
                                        <abbr title={title} className="no-underline">
                                            {label}
                                        </abbr>
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="tabular-nums">
                            {style.map((s) =>
                                s.hands < fewHands ? (
                                    <tr key={s.name} className="text-subtle">
                                        <td className="font-semibold">{s.name}</td>
                                        <td colSpan={styleColumns.length} className="text-right">
                                            {fewHands - s.hands === 1 ? "1 more hand" : `${fewHands - s.hands} more hands`}
                                        </td>
                                    </tr>
                                ) : (
                                    <tr key={s.name} className="text-fg">
                                        <td className="font-semibold">{s.name}</td>
                                        <td className="text-right">{s.vpip}</td>
                                        <td className="text-right">{s.pfr}</td>
                                        <td className="text-right">{s.threeBet}</td>
                                        <td className="text-right">{s.af}</td>
                                        <td className="text-right">{s.wtsd}</td>
                                        <td className="text-right">{s.wsd}</td>
                                        <td className="text-right">{s.bb100}</td>
                                    </tr>
                                )
                            )}
                        </tbody>
                    </table>
                </div>
            )}
            {best.length > 0 && (
                <div className="text-xs">
                    <h3 className="text-subtle">best hand shown</h3>
                    <ul aria-label="best hands shown">
                        {best.map((s) => (
                            <li key={s.name} className="flex gap-2">
                                <span className="font-semibold text-fg">{s.name}</span>{" "}
                                <span className="text-muted">{s.bestHand}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            <p className="text-xs text-subtle">
                {hands.length === 1 ? "1 hand" : `${hands.length} hands`}
                {hands[0].hand > 1 && ` · from hand ${hands[0].hand}`}
                {perHour > 0 && ` · ${perHour}/hr`}
            </p>
        </div>
    );
}

// minRateHours is how long hands are played before the hourly rate shows
const minRateHours = 5 / 60;

