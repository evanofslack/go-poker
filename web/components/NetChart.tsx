import { useEffect, useRef, useState } from "react";
import classNames from "classnames";
import { HandStats } from "../interfaces/index";
import { signed } from "../lib/ledger";
import { chartValues, niceTicks, seriesColor } from "../lib/stats";

const pad = { top: 8, right: 8, bottom: 16, left: 38 };
const labelWidth = 44; // room for names at the line ends

// useSize tracks an element's size, so the chart draws at its real size
function useSize() {
    const ref = useRef<HTMLDivElement>(null);
    const [size, setSize] = useState({ width: 0, height: 0 });
    useEffect(() => {
        if (!ref.current) return;
        const ro = new ResizeObserver(([e]) => setSize({ width: e.contentRect.width, height: e.contentRect.height }));
        ro.observe(ref.current);
        return () => ro.disconnect();
    }, []);
    return [ref, size] as const;
}

// NetChart is a line per player of their net, chips won or lost, at the end of
// each hand, with a crosshair and tooltip on hover. Without a height it fills
// its parent.
export default function NetChart({ hands, names, height }: { hands: HandStats[]; names: string[]; height?: number }) {
    const [ref, size] = useSize();
    const [hover, setHover] = useState<number | null>(null);
    const { net, value } = chartValues(hands);
    const width = size.width;
    const h = height ?? size.height;
    const labelled = names.length <= 4;
    const right = pad.right + (labelled ? labelWidth : 0);
    const plotW = Math.max(width - pad.left - right, 1);
    const plotH = Math.max(h - pad.top - pad.bottom, 1);
    const values = hands.flatMap((hand) => names.map((n) => value(hand, n)).filter((v) => v !== undefined));
    // nets keep 0 in view; stacks fit their range, so small swings show
    const min = net ? Math.min(0, ...values) : Math.min(...values);
    const max = net ? Math.max(0, ...values) : Math.max(...values);
    const ticks = niceTicks(min, max);
    const lo = ticks[0];
    const hi = ticks[ticks.length - 1];
    const x = (i: number) => pad.left + (hands.length === 1 ? plotW / 2 : (i * plotW) / (hands.length - 1));
    const y = (v: number) => pad.top + plotH - ((v - lo) / (hi - lo)) * plotH;
    const show = (v: number) => (net ? signed(v) : String(v));

    // a player's line breaks over hands they weren't seated for
    const path = (name: string) => {
        let d = "";
        let pen = false;
        hands.forEach((hand, i) => {
            const v = value(hand, name);
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
            if (value(hands[i], name) !== undefined) return i;
        }
        return -1;
    };
    const onMove = (e: React.PointerEvent<SVGRectElement>) => {
        const box = e.currentTarget.getBoundingClientRect();
        const frac = (e.clientX - box.left) / box.width;
        setHover(Math.min(hands.length - 1, Math.max(0, Math.round(frac * (hands.length - 1)))));
    };
    const hovered = hover === null ? null : hands[hover];

    // end labels, nudged apart so they don't overlap
    const labels = labelled
        ? names
              .map((n) => ({ n, at: lastSeen(n) }))
              .filter((l) => l.at >= 0)
              .map((l) => ({ ...l, y: y(value(hands[l.at], l.n)!) }))
              .sort((a, b) => a.y - b.y)
        : [];
    for (let i = 1; i < labels.length; i++) {
        labels[i].y = Math.max(labels[i].y, labels[i - 1].y + 11);
    }

    return (
        <div className="flex h-full flex-col gap-1">
            {/* short tables label the line ends instead */}
            {!labelled && (
                <ul className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted" aria-label="chart legend">
                    {names.map((n, i) => (
                        <li key={n} className="flex items-center gap-1">
                            <span className="inline-block h-0.5 w-3 shrink-0 rounded-full" style={{ background: seriesColor(i) }} />
                            {n}
                        </li>
                    ))}
                </ul>
            )}
            <div ref={ref} className={classNames("relative rounded-md bg-panel", !height && "min-h-0 flex-1")} style={{ height }}>
                {width > 0 && h > 0 && (
                    <svg
                        width={width}
                        height={h}
                        className="absolute inset-0"
                        role="img"
                        aria-label={net ? "net over time per player" : "chips over time per player"}
                    >
                        {ticks.map((t) => (
                            <g key={t}>
                                <line
                                    x1={pad.left}
                                    x2={pad.left + plotW}
                                    y1={y(t)}
                                    y2={y(t)}
                                    stroke={net && t === 0 ? "var(--color-subtle)" : "var(--color-raised)"}
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
                                    {show(t)}
                                </text>
                            </g>
                        ))}
                        <text x={pad.left} y={h - 4} fontSize={10} fill="var(--color-subtle)">
                            hand {hands[0].hand}
                        </text>
                        {hands.length > 1 && (
                            <text x={pad.left + plotW} y={h - 4} fontSize={10} textAnchor="end" fill="var(--color-subtle)">
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
                            names.map((n, i) => {
                                const v = value(hovered, n);
                                return v === undefined ? null : (
                                    <circle
                                        key={n}
                                        cx={x(hover!)}
                                        cy={y(v)}
                                        r={4}
                                        fill={seriesColor(i)}
                                        stroke="var(--color-panel)"
                                        strokeWidth={2}
                                    />
                                );
                            })}
                        <rect
                            x={pad.left}
                            y={0}
                            width={plotW}
                            height={h}
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
                            .map((n, i) => ({ n, i, v: value(hovered, n) }))
                            .filter((r): r is { n: string; i: number; v: number } => r.v !== undefined)
                            .sort((a, b) => b.v - a.v)
                            .map(({ n, i, v }) => (
                                <p key={n} className="flex items-center gap-1 tabular-nums">
                                    <span className="inline-block h-2 w-2 shrink-0 rounded-full" style={{ background: seriesColor(i) }} />
                                    {n}
                                    <span className={classNames("ml-auto pl-2", net && v > 0 && "text-gain", net && v < 0 && "text-loss")}>
                                        {show(v)}
                                    </span>
                                    {hovered.won[n] ? <span className="text-subtle">won {hovered.won[n]}</span> : null}
                                </p>
                            ))}
                    </div>
                )}
            </div>
        </div>
    );
}
