import { ReactNode, useContext } from "react";
import { FiCopy } from "react-icons/fi";
import { AppContext } from "../../providers/AppStore";
import { copyInvite } from "../../lib/invite";

const chips = (n: number) => n.toLocaleString("en");
const seconds = (ms: number) => `${ms / 1000}s`;

const keys: [string, string][] = [
    ["F", "fold"],
    ["C", "check or call"],
    ["R / B", "bet or raise"],
    ["↑ ↓", "change a bet by a big blind"],
    ["Enter", "place the bet"],
    ["Esc", "close the bet"],
];

// Settings shows how the host set up the table. They're fixed once it exists.
export default function Settings() {
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.view;
    const table = appState.table;
    if (!game || !table) return null;
    const s = game.settings;

    const rows: [string, ReactNode][] = [
        [
            "table",
            <span className="flex items-center justify-end gap-1">
                <span className="truncate">{table}</span>
                <button
                    className="flex shrink-0 items-center gap-1 rounded-md px-1.5 text-subtle hover:text-fg"
                    title="copy invite link"
                    aria-label="copy invite link"
                    onClick={() => copyInvite(table, appState.password, dispatch)}
                >
                    <FiCopy /> invite
                </button>
            </span>,
        ],
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
                        <dd className="ml-auto min-w-0 text-right text-fg tabular-nums">{value}</dd>
                    </div>
                ))}
            </dl>
            {/* only for a mouse and keyboard */}
            <div className="mt-4 hidden text-sm pointer-fine:block">
                <label className="flex items-center gap-2 py-1.5 text-fg">
                    <input
                        type="checkbox"
                        className="size-4 accent-accent"
                        checked={appState.shortcuts}
                        onChange={(e) => dispatch({ type: "setShortcuts", payload: e.target.checked })}
                    />
                    keyboard shortcuts
                </label>
                <dl aria-label="keyboard shortcuts" className={appState.shortcuts ? "" : "opacity-50"}>
                    {keys.map(([key, does]) => (
                        <div key={key} className="flex gap-2 border-b border-raised py-1.5 last:border-b-0">
                            <dt>
                                <kbd className="rounded border border-line px-1.5 font-sans text-fg">{key}</kbd>
                            </dt>
                            <dd className="ml-auto text-right text-subtle">{does}</dd>
                        </div>
                    ))}
                </dl>
                <p className="py-1.5 text-subtle">Before your turn, F and C pick an action instead.</p>
            </div>
        </div>
    );
}
