import { ReactNode, useContext } from "react";
import { AppContext } from "../../providers/AppStore";

const chips = (n: number) => n.toLocaleString("en");
const seconds = (ms: number) => `${ms / 1000}s`;

// Settings shows how the host set up the table. They're fixed once it exists.
export default function Settings() {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game) return null;
    const s = game.settings;

    const rows: [string, ReactNode][] = [
        ["game", "no limit texas hold'em"],
        ["stakes", `${chips(s.sb)}/${chips(s.bb)}`],
        [
            "buy-in",
            <>
                {chips(s.minBuy)}–{chips(s.maxBuy)}
                <span className="text-subtle">
                    {" "}
                    ({Math.round(s.minBuy / s.bb)}–{Math.round(s.maxBuy / s.bb)} bb)
                </span>
            </>,
        ],
        ["seats", s.seats],
        ["turn timer", s.turnTime ? seconds(s.turnTime) : "off"],
        ["next hand", seconds(s.nextHandDelay)],
        ["listing", s.public ? "public" : "private"],
        ["password", game.hasPassword ? "on" : "off"],
        ["rebuys", s.approveRebuys ? "host approves" : "automatic"],
        ["host", game.host],
    ];

    return (
        <div className="h-full overflow-auto bg-surface p-2">
            <dl aria-label="table settings" className="text-sm">
                {rows.map(([label, value]) => (
                    <div key={label} className="flex gap-2 border-b border-raised py-1.5 last:border-b-0">
                        <dt className="text-subtle">{label}</dt>
                        <dd className="ml-auto text-right text-fg tabular-nums">{value}</dd>
                    </div>
                ))}
            </dl>
        </div>
    );
}
