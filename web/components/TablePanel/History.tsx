import { useContext, useState } from "react";
import classNames from "classnames";
import { getHands } from "../../actions/actions";
import { useSocket } from "../../hooks/useSocket";
import { HandPot, HandRecord } from "../../interfaces/index";

// Card is a card as hand history spells it, like "Ks"
type Card = string;
import { actionText, netText, streets, winnersText } from "../../lib/history";
import { AppContext } from "../../providers/AppStore";
import { Button, Checkbox } from "@mantine/core";

const suits: Record<string, { glyph: string; name: string; color: string }> = {
    s: { glyph: "♠", name: "spades", color: "text-spade" },
    h: { glyph: "♥", name: "hearts", color: "text-heart" },
    d: { glyph: "♦", name: "diamonds", color: "text-diamond" },
    c: { glyph: "♣", name: "clubs", color: "text-club" },
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
            className={classNames("rounded-sm bg-card px-0.5 font-semibold leading-tight", suit.color)}
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
                <Checkbox
                    size="xs"
                    label="hands I played"
                    checked={mine}
                    onChange={(e) => setMine(e.currentTarget.checked)}
                />
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
                        <Button variant="subtle" size="compact-xs" onClick={() => getHands(socket, history[0].hand)}>
                            load older hands
                        </Button>
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

// potName says which pot p is: the only one, the main pot or a side pot.
// With the board run out more than once, each pot shows up once a run.
function potName(pots: HandPot[], p: HandPot) {
    const runs = new Set(pots.map((q) => q.run ?? 0)).size;
    if (pots.length === runs) return "pot";
    return pots.filter((q) => (q.run ?? 0) === (p.run ?? 0)).indexOf(p) === 0 ? "main pot" : "side pot";
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
            {(r.boards ?? []).slice(1).map((b, i) => (
                <p key={i} className="flex items-center gap-1.5 text-subtle" data-history-run>
                    run {i + 2}
                    <Cards cards={b} />
                </p>
            ))}
            <ul aria-label="pots">
                {(r.pots ?? []).map((p, i, pots) => (
                    <li key={i}>
                        {r.boards && `run ${(p.run ?? 0) + 1}: `}
                        {potName(pots, p)} {p.amount} ·{" "}
                        <span className="text-fg">{p.winners.join(", ")}</span>
                        {p.hand && `, ${p.hand}`}
                    </li>
                ))}
            </ul>
            {r.payments && r.payments.length > 0 && (
                <ul aria-label="side games">
                    {r.payments.map((p, i) => (
                        <li key={i}>
                            <span className="text-fg">{p.from}</span> paid <span className="text-fg">{p.to}</span>{" "}
                            {p.amount} for the {p.kind}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}
