import { useContext, useEffect, useRef, useState } from "react";
import { HandStats } from "../../interfaces/index";
import { fewHands, names, playerStats, seriesColor, styleStats } from "../../lib/stats";
import { AppContext } from "../../providers/AppStore";

const height = 130;
const pad = { top: 8, right: 8, bottom: 16, left: 34 };
const labelWidth = 44; // room for names at the line ends

// Stats shows each player's chips over the hands played, and their totals.
export default function Stats() {
    const { appState } = useContext(AppContext);
    const hands = appState.hands;
    if (hands.length === 0) {
        return <p className="p-2 text-sm text-subtle">stats start after the first hand</p>;
    }
    const all = names(hands);
    const stats = playerStats(hands);
    const style = styleStats(appState.totals, all, appState.game?.settings.bb ?? 0);
    const best = style.filter((s) => s.bestHand);

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
                <div className="overflow-x-auto">
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
                {style.some((s) => s.hands < fewHands) && ` · play style shows after ${fewHands} hands`}
            </p>
        </div>
    );
}

// styleColumns label the play style table, with what each one means on hover
const styleColumns = [
    ["VPIP", "voluntarily put money in: hands they called or raised before the flop"],
    ["PFR", "preflop raise: hands they raised before the flop"],
    ["3B", "3-bet: re-raised a raise before the flop, when they had the chance"],
    ["AF", "aggression factor: bets and raises for each call"],
    ["WTSD", "went to showdown, of the hands they saw a flop in"],
    ["W$SD", "won at showdown, of the showdowns they went to"],
    ["bb/100", "big blinds won per 100 hands"],
];

// useWidth tracks an element's width, so the chart draws at its real size
function useWidth() {
    const ref = useRef<HTMLDivElement>(null);
    const [width, setWidth] = useState(0);
    useEffect(() => {
        if (!ref.current) return;
        const ro = new ResizeObserver(([e]) => setWidth(e.contentRect.width));
        ro.observe(ref.current);
        return () => ro.disconnect();
    }, []);
    return [ref, width] as const;
}

// StackChart is a line per player of their stack at the end of each hand,
// with a crosshair and tooltip on hover
function StackChart({ hands, names }: { hands: HandStats[]; names: string[] }) {
    const [ref, width] = useWidth();
    const [hover, setHover] = useState<number | null>(null);
    const labelled = names.length <= 4;
    const right = pad.right + (labelled ? labelWidth : 0);
    const plotW = Math.max(width - pad.left - right, 1);
    const plotH = height - pad.top - pad.bottom;
    // fit the stacks' range, not from 0, so small swings show
    const values = hands.flatMap((h) => Object.values(h.stacks));
    const spread = Math.max(Math.max(...values) - Math.min(...values), 10);
    const lo = Math.max(0, Math.floor(Math.min(...values) - spread * 0.1));
    const hi = Math.ceil(Math.max(...values) + spread * 0.1);
    const x = (i: number) => pad.left + (hands.length === 1 ? plotW / 2 : (i * plotW) / (hands.length - 1));
    const y = (v: number) => pad.top + plotH - ((v - lo) / (hi - lo)) * plotH;

    // a player's line breaks over hands they weren't seated for
    const path = (name: string) => {
        let d = "";
        let pen = false;
        hands.forEach((h, i) => {
            const v = h.stacks[name];
            if (v === undefined) {
                pen = false;
                return;
            }
            d += `${pen ? "L" : "M"}${x(i).toFixed(1)},${y(v).toFixed(1)}`;
            pen = true;
        });
        return d;
    };
    const lastSeen = (name: string) => {
        for (let i = hands.length - 1; i >= 0; i--) {
            if (hands[i].stacks[name] !== undefined) return i;
        }
        return -1;
    };
    const onMove = (e: React.PointerEvent<SVGRectElement>) => {
        const box = e.currentTarget.getBoundingClientRect();
        const frac = (e.clientX - box.left) / box.width;
        setHover(Math.min(hands.length - 1, Math.max(0, Math.round(frac * (hands.length - 1)))));
    };
    const hovered = hover === null ? null : hands[hover];
    const ticks = [lo, Math.round((lo + hi) / 2), hi];

    // end labels, nudged apart so they don't overlap
    const labels = labelled
        ? names
              .map((n) => ({ n, at: lastSeen(n) }))
              .filter((l) => l.at >= 0)
              .map((l) => ({ ...l, y: y(hands[l.at].stacks[l.n]) }))
              .sort((a, b) => a.y - b.y)
        : [];
    for (let i = 1; i < labels.length; i++) {
        labels[i].y = Math.max(labels[i].y, labels[i - 1].y + 11);
    }

    return (
        <div ref={ref} className="relative rounded-md bg-panel" style={{ height }}>
            {width > 0 && (
                <svg width={width} height={height} role="img" aria-label="chips over time per player">
                    {ticks.map((t) => (
                        <g key={t}>
                            <line
                                x1={pad.left}
                                x2={pad.left + plotW}
                                y1={y(t)}
                                y2={y(t)}
                                stroke="var(--color-raised)"
                                strokeWidth={1}
                            />
                            <text
                                x={pad.left - 4}
                                y={y(t)}
                                dy="0.32em"
                                textAnchor="end"
                                fontSize={10}
                                fill="var(--color-subtle)"
                            >
                                {t}
                            </text>
                        </g>
                    ))}
                    <text x={pad.left} y={height - 4} fontSize={10} fill="var(--color-subtle)">
                        hand {hands[0].hand}
                    </text>
                    {hands.length > 1 && (
                        <text x={pad.left + plotW} y={height - 4} fontSize={10} textAnchor="end" fill="var(--color-subtle)">
                            {hands[hands.length - 1].hand}
                        </text>
                    )}
                    {hover !== null && (
                        <line
                            x1={x(hover)}
                            x2={x(hover)}
                            y1={pad.top}
                            y2={pad.top + plotH}
                            stroke="var(--color-muted)"
                            strokeWidth={1}
                        />
                    )}
                    {names.map((n, i) => (
                        <path
                            key={n}
                            d={path(n)}
                            fill="none"
                            stroke={seriesColor(i)}
                            strokeWidth={2}
                            strokeLinejoin="round"
                            strokeLinecap="round"
                        />
                    ))}
                    {labels.map(({ n, at, y }) => (
                        <text key={n} x={x(at) + 6} y={y} dy="0.32em" fontSize={10} fill="var(--color-muted)">
                            {n}
                        </text>
                    ))}
                    {hovered &&
                        names.map((n, i) =>
                            hovered.stacks[n] === undefined ? null : (
                                <circle
                                    key={n}
                                    cx={x(hover!)}
                                    cy={y(hovered.stacks[n])}
                                    r={4}
                                    fill={seriesColor(i)}
                                    stroke="var(--color-panel)"
                                    strokeWidth={2}
                                />
                            )
                        )}
                    <rect
                        x={pad.left}
                        y={0}
                        width={plotW}
                        height={height}
                        fill="transparent"
                        onPointerMove={onMove}
                        onPointerDown={onMove}
                        onPointerLeave={() => setHover(null)}
                    />
                </svg>
            )}
            {hovered && (
                <div
                    className="pointer-events-none absolute top-1 z-10 rounded-md bg-surface px-2 py-1 text-xs text-fg shadow-lg"
                    style={x(hover!) > width / 2 ? { right: width - x(hover!) + 8 } : { left: x(hover!) + 8 }}
                    role="tooltip"
                >
                    <p className="text-subtle">hand {hovered.hand}</p>
                    {names
                        .map((n, i) => ({ n, i, v: hovered.stacks[n] }))
                        .filter((r) => r.v !== undefined)
                        .sort((a, b) => b.v - a.v)
                        .map(({ n, i, v }) => (
                            <p key={n} className="flex items-center gap-1 tabular-nums">
                                <span className="inline-block h-2 w-2 shrink-0 rounded-full" style={{ background: seriesColor(i) }} />
                                {n} <span className="ml-auto pl-2">{v}</span>
                                {hovered.won[n] ? <span className="text-subtle">won {hovered.won[n]}</span> : null}
                            </p>
                        ))}
                </div>
            )}
        </div>
    );
}
