import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { removeBot, takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { PlayerView, View } from "../interfaces/index";
import Card, { MiniCards, cardSize } from "./Card";
import BuyIn from "./BuyIn";
import Dialog from "./Dialog";
import AddBot from "./AddBot";
import Countdown from "./Countdown";
import SeatMenu from "./SeatMenu";
import { blind, isWinner, madeHand, winningCards, winningHand, winnings } from "../lib/hands";
import { phone, useMedia } from "../hooks/useMedia";
import classNames from "classnames";

const plate =
    "w-24 rounded-xl px-2 py-1 text-center sm:w-32 lg:w-40 lg:py-1.5 short:w-24 short:py-0.5";

function plateStyle(player: PlayerView, view: View) {
    const action = view.betting && player.seat === view.toAct;
    const counting = action && view.turnEndsIn > 0;
    const winner = !view.betting && isWinner(view, player.seat);
    return classNames(plate, "relative", {
        "bg-zinc-100 text-zinc-900 shadow-[0_0_40px_2px_rgba(255,255,255,0.3)]": action,
        "bg-amber-200 text-zinc-900 shadow-[0_0_60px_20px_rgba(100,98,92,0.3)]": winner,
        "bg-panel text-fg": !action && !winner,
        // sitting out or out of chips
        "opacity-50": player.sittingOut && !action && !winner,
        // folded this hand
        "opacity-60": !player.sittingOut && !player.in && view.street !== "",
        // clear the cards that overlap the top of the plate
        "pt-2.5 short:py-0.5 short:pl-3": true,
        // room for the countdown along the bottom
        "pb-2.5 short:pb-2 lg:pb-3": counting,
    });
}

const tag =
    "absolute -bottom-3 left-1/2 z-20 -translate-x-1/2 animate-pop rounded-full px-2 text-xs font-semibold whitespace-nowrap shadow-md sm:text-sm short:-bottom-2 short:text-xs";

// ResultTag shows what a player won and with what once the hand is over, and
// on your own seat, your best hand while you're in one
function ResultTag({ player, view, own }: { player: PlayerView; view: View; own: boolean }) {
    const won = winnings(view, player.seat);
    if (won) {
        const hand = winningHand(view, player.seat);
        return (
            <span className={classNames(tag, "bg-amber-400 text-zinc-900")} data-won>
                won {won}
                {hand && (
                    <>
                        {" · "}
                        {/* just the kind of hand, like "full house", with the rest on hover */}
                        <span data-hand title={hand}>
                            {hand.replace(/^an? /, "").split(",")[0]}
                        </span>
                    </>
                )}
            </span>
        );
    }
    const made = own && player.in && view.street !== "" ? madeHand([...player.cards, ...view.board]) : "";
    if (!made) return null;
    return (
        <span key={made} className={classNames(tag, "border border-line bg-panel font-normal text-fg")} data-made-hand>
            {made}
        </span>
    );
}

// BotIcon marks a bot's seat, with its level on hover
function BotIcon({ level }: { level: string }) {
    return (
        <span
            className="ml-1 inline-flex shrink-0 opacity-70"
            title={`${level} bot`}
            role="img"
            aria-label={`${level} bot`}
            data-bot-badge
        >
            <svg viewBox="0 0 16 16" className="h-3.5 w-3.5 sm:h-4 sm:w-4" aria-hidden>
                <path d="M8 1v3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                <rect x="2.25" y="4.25" width="11.5" height="9.5" rx="2.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
                <circle cx="6" cy="9" r="1.25" fill="currentColor" />
                <circle cx="10" cy="9" r="1.25" fill="currentColor" />
            </svg>
        </span>
    );
}

type playerSeatProps = {
    player: PlayerView;
    view: View;
    own: boolean;
    host: boolean; // the viewer is the host
};

// Badges put the dealer button and blind on a phone's seat plate, where there's
// no room on the felt to set them in front of the seat
function Badges({ player, view, own }: { player: PlayerView; view: View; own: boolean }) {
    const dealer = view.running && view.button === player.seat;
    const label = view.running ? blind(view, player.seat) : "";
    if (!dealer && !label) return null;
    return (
        // your own cards cover the top of your plate and your hand is under it, so
        // yours go beside it
        <span
            className={classNames(
                "absolute z-20 flex gap-0.5",
                own ? "top-1/2 right-full mr-1 -translate-y-1/2" : "-top-2 -left-2"
            )}
            data-badges
        >
            {dealer && (
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-white text-xs font-bold text-purple-800 shadow-md">
                    D
                </span>
            )}
            {label && (
                <span className="flex h-5 items-center rounded-full bg-zinc-300 px-1.5 text-xs font-bold text-zinc-800 shadow-md">
                    {label}
                </span>
            )}
        </span>
    );
}

function PlayerSeat({ player, view, own, host }: playerSeatProps) {
    const socket = useSocket();
    const onPhone = useMedia(phone);
    const winning = winningCards(view);
    const bot = player.bot;
    // on a phone, other players' face-down cards shrink onto the plate, and only
    // cards you can read, like at showdown, take the full size
    const mini = onPhone && !own && player.dealt && player.cards.every((c) => !c);
    return (
        <div className="flex flex-col items-center short:flex-row">
            <div
                className={classNames(
                    "relative z-10 -mb-2 flex gap-0.5 short:-mr-2 short:mb-0",
                    cardSize.h,
                    mini && "hidden"
                )}
            >
                {player.dealt &&
                    player.cards.map((c, i) => (
                        <Card
                            key={i}
                            card={c}
                            placeholder={false}
                            // everyone is out between hands; cards shown then reached showdown
                            folded={!player.in && !(view.street === "" && c)}
                            hidden={!c}
                            highlight={winning.has(c)}
                        />
                    ))}
            </div>
            <div className={plateStyle(player, view)}>
                <p
                    className={classNames(
                        "truncate text-sm leading-tight sm:text-base short:text-xs",
                        bot && "flex items-center justify-center"
                    )}
                >
                    {/* a bot's long name gives way to its icon */}
                    {bot ? <span className="truncate">{player.name}</span> : player.name}
                    {bot && <BotIcon level={bot} />}
                </p>

                <p className="text-sm leading-tight font-semibold sm:text-lg short:text-sm">
                    {player.stack}
                </p>
                {view.betting && view.toAct === player.seat && (
                    <span className="sr-only">to act</span>
                )}
                {player.sittingOut && (
                    <span className="absolute -top-2.5 left-1/2 z-20 -translate-x-1/2 rounded-full bg-badge px-2 text-xs whitespace-nowrap text-fg">
                        {player.stack > 0 ? "sitting out" : "out of chips"}
                    </span>
                )}
                <ResultTag player={player} view={view} own={own} />
                {onPhone && <Badges player={player} view={view} own={own} />}
                {mini && player.in && <MiniCards count={player.cards.length} />}
                {own && <SeatMenu player={player} view={view} />}
                {bot && host && socket && !view.ended && (
                    // on the top corner, clear of the name, icon and result tag
                    <button
                        className="absolute -top-2 -right-2 z-20 flex h-5 w-5 items-center justify-center rounded-full bg-raised text-sm leading-none text-muted hover:text-fg"
                        title="remove bot"
                        aria-label={`remove ${player.name}`}
                        onClick={() => removeBot(socket, player.seat)}
                    >
                        ×
                    </button>
                )}
                {view.betting && view.toAct === player.seat && view.turnEndsIn > 0 && (
                    <Countdown endsIn={view.turnEndsIn} total={view.settings.turnTime} />
                )}
            </div>
        </div>
    );
}

// OpenSeat lets someone without a seat sit down, after choosing a buy-in. The
// host can sit a bot there instead.
function OpenSeat({ id, view, host }: { id: number; view: View; host: boolean }) {
    const socket = useSocket();
    const [open, setOpen] = useState<"sit" | "bot" | null>(null);
    const close = () => setOpen(null);

    return (
        <>
            <button
                className={classNames(
                    plate,
                    "border-2 border-dashed border-line bg-raised/60 text-fg hover:bg-raised"
                )}
                onClick={() => setOpen("sit")}
            >
                <span className="text-lg sm:text-2xl">{id}</span>
                <span className="block text-xs opacity-70 sm:text-sm">Open</span>
            </button>
            {open === "sit" && (
                <Dialog title={`sit in seat ${id}`} onClose={close}>
                    <BuyIn
                        min={view.settings.minBuy}
                        max={view.settings.maxBuy}
                        initial={100 * view.settings.bb}
                        label="sit down"
                        onSubmit={(amount) => {
                            if (socket) {
                                takeSeat(socket, id, amount);
                            }
                            close();
                        }}
                        onCancel={close}
                    />
                    {host && (
                        <button
                            className="mt-2 px-3 text-sm text-subtle underline hover:text-fg"
                            onClick={() => setOpen("bot")}
                        >
                            add a bot here instead
                        </button>
                    )}
                </Dialog>
            )}
            {open === "bot" && (
                <Dialog title={`add a bot to seat ${id}`} onClose={close}>
                    <AddBot seat={id} view={view} onDone={close} />
                </Dialog>
            )}
        </>
    );
}

// BotSeat is an empty seat as the seated host sees it, for adding a bot
function BotSeat({ id, view }: { id: number; view: View }) {
    const [open, setOpen] = useState(false);
    return (
        <>
            <button
                className={classNames(
                    plate,
                    "border-2 border-dashed border-zinc-700 bg-raised/40 text-muted hover:bg-raised hover:text-fg"
                )}
                onClick={() => setOpen(true)}
            >
                <span className="text-lg sm:text-2xl">{id}</span>
                <span className="block text-xs opacity-70 sm:text-sm">+ bot</span>
            </button>
            {open && (
                <Dialog title={`add a bot to seat ${id}`} onClose={() => setOpen(false)}>
                    <AddBot seat={id} view={view} onDone={() => setOpen(false)} />
                </Dialog>
            )}
        </>
    );
}

export default function Seat({ player, id }: { player: PlayerView | null; id: number }) {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!view) return null;
    const host = view.host === appState.username;

    if (player) {
        // the server only sends cards this client may see
        return <PlayerSeat player={player} view={view} own={player.seat === view.you} host={host} />;
    }
    // nobody sits down, and no bots join, once the session has ended
    if (view.ended) return null;
    if (!view.you) {
        return <OpenSeat id={id} view={view} host={host} />;
    }
    if (host) {
        return <BotSeat id={id} view={view} />;
    }
    // seated players see empty seats faintly until the game starts
    if (view.running) return null;
    return (
        <div className={classNames(plate, "bg-raised py-2 text-lg text-muted opacity-20")}>
            {id}
        </div>
    );
}
