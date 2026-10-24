import { useContext, useState } from "react";
import classNames from "classnames";
import { getHands } from "../../actions/actions";
import { useSocket } from "../../hooks/useSocket";
import { HandRecord } from "../../interfaces/index";

// Card is a card as hand history spells it, like "Ks"
type Card = string;
import { actionText, netText, streets, winnersText } from "../../lib/history";
import { AppContext } from "../../providers/AppStore";

const suits: Record<string, { glyph: string; name: string; color: string }> = {
    s: { glyph: "♠", name: "spades", color: "text-black" },
    h: { glyph: "♥", name: "hearts", color: "text-red-700" },
    d: { glyph: "♦", name: "diamonds", color: "text-blue-700" },
    c: { glyph: "♣", name: "clubs", color: "text-green-700" },
};
const ranks: Record<string, string> = { T: "ten", J: "jack", Q: "queen", K: "king", A: "ace" };

// MiniCard is a card small enough for a line of text, like "K♠"
function MiniCard({ card }: { card: Card }) {
    const suit = suits[card[1]?.toLowerCase()];
    if (!suit) return null;
    return (
        <span
            role="img"
            aria-label={`${ranks[card[0]] ?? card[0]} of ${suit.name}`}
            className={classNames("rounded-sm bg-white px-0.5 font-semibold leading-tight", suit.color)}
        >
            {card[0]}
            {suit.glyph}
        </span>
    );
}

function Cards({ cards }: { cards?: Card[] | null }) {
    if (!cards?.length) return null;
    return (
        <span className="inline-flex gap-0.5">
            {cards.map((c) => (
                <MiniCard key={c} card={c} />
            ))}
        </span>
    );
}

function Net({ net }: { net: number }) {
    return <span className={classNames("tabular-nums", net > 0 && "text-gain", net < 0 && "text-loss")}>{netText(net)}</span>;
}

// History lists the hands played, newest first. Each opens to show the
// actions street by street. You only see others' cards if they were shown.
export default function History() {
    const { appState } = useContext(AppContext);
    const socket = useSocket();
    const me = appState.username;
    const [mine, setMine] = useState(false);
    // undefined keeps the newest hand open as new ones come in
    const [open, setOpen] = useState<number | null | undefined>(undefined);

    const history = appState.history;
    if (history.length === 0) {
        return <p className="p-2 text-sm text-subtle">hands show up here once they're played</p>;
    }
    const hands = history.filter((r) => !mine || r.players.some((p) => p.name === me)).reverse();
    const newest = history[history.length - 1].hand;
    const isOpen = (hand: number) => (open === undefined ? hand === newest : open === hand);

    return (
        <div className="flex h-full flex-col bg-surface">
            <div className="flex items-center gap-2 border-b border-raised px-2 py-1.5 text-xs">
                <label className="flex cursor-pointer items-center gap-1.5 text-muted">
                    <input type="checkbox" checked={mine} onChange={(e) => setMine(e.target.checked)} />
                    hands I played
                </label>
                <span className="ml-auto text-subtle">
                    {history.length === 1 ? "1 hand" : `${history.length} hands`}
                </span>
            </div>
            <ol className="min-h-0 flex-1 overflow-auto p-2" aria-label="hand history">
                {hands.map((r) => (
                    <Hand
                        key={r.hand}
                        record={r}
                        me={me}
                        open={isOpen(r.hand)}
                        onToggle={() => setOpen(isOpen(r.hand) ? null : r.hand)}
                    />
                ))}
                {hands.length === 0 && <li className="text-sm text-subtle">you haven't played a hand yet</li>}
                {appState.moreHistory && socket && (
                    <li className="pt-2 text-center">
                        <button
                            className="rounded-md px-2 py-0.5 text-xs text-muted hover:bg-raised hover:text-fg"
                            onClick={() => getHands(socket, history[0].hand)}
                        >
                            load older hands
                        </button>
                    </li>
                )}
            </ol>
        </div>
    );
}

function Hand({ record: r, me, open, onToggle }: { record: HandRecord; me: string | null; open: boolean; onToggle: () => void }) {
    const you = r.players.find((p) => p.name === me);
    return (
        <li className="border-b border-raised py-1.5 text-xs last:border-b-0">
            <button
                className="flex w-full items-center gap-2 text-left"
                aria-expanded={open}
                aria-label={`hand ${r.hand}`}
                onClick={onToggle}
            >
                <span className="w-8 shrink-0 text-subtle tabular-nums">#{r.hand}</span>
                <span className="min-w-0 flex-1 truncate text-fg">{winnersText(r)}</span>
                {you && <Cards cards={you.cards} />}
                {you && (
                    <span className="w-12 text-right">
                        <Net net={you.net} />
                    </span>
                )}
            </button>
            {open && <HandDetail record={r} me={me} />}
        </li>
    );
}

function HandDetail({ record: r, me }: { record: HandRecord; me: string | null }) {
    return (
        <div className="mt-1.5 flex flex-col gap-1.5 pl-10 text-muted">
            <ul aria-label="players">
                {r.players.map((p) => (
                    <li key={p.name} className="flex items-center gap-1.5">
                        <span className={classNames("truncate", p.name === me ? "font-semibold text-fg" : "text-fg")}>
                            {p.name}
                        </span>
                        {p.name === r.dealer && <span className="text-subtle">dealer</span>}
                        <span className="text-subtle tabular-nums">{p.stack}</span>
                        <Cards cards={p.cards} />
                        {p.hand && <span className="truncate">{p.hand}</span>}
                        {p.showdown && !p.shown && <span className="text-subtle">mucked</span>}
                        <span className="ml-auto">
                            <Net net={p.net} />
                        </span>
                    </li>
                ))}
            </ul>
            {streets(r).map((s) => (
                <div key={s.name}>
                    <p className="flex items-center gap-1.5 text-subtle">
                        {s.name}
                        <Cards cards={s.cards} />
                    </p>
                    <ul>
                        {s.actions.map((a, i) => (
                            <li key={i}>
                                <span className="text-fg">{a.name}</span> {actionText(a)}
                            </li>
                        ))}
                    </ul>
                </div>
            ))}
            <ul aria-label="pots">
                {(r.pots ?? []).map((p, i, pots) => (
                    <li key={i}>
                        {pots.length === 1 ? "pot" : i === 0 ? "main pot" : "side pot"} {p.amount} ·{" "}
                        <span className="text-fg">{p.winners.join(", ")}</span>
                        {p.hand && `, ${p.hand}`}
                    </li>
                ))}
            </ul>
        </div>
    );
}
