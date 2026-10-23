import classNames from "classnames";
import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { PlayerView, View } from "../interfaces";
import { at, besideSeat, markerAt, ringPoint, seatAngle, tallPoint } from "../lib/seats";
import { blind } from "../lib/hands";
import { phone, tall, useMedia } from "../hooks/useMedia";
import Seat from "./Seat";
import Pot from "./Pot";
import CommunityCards from "./CommunityCards";
import TableHint from "./TableHint";
import BuyRequests from "./BuyRequests";

// half a seat's size at each breakpoint, which keeps the seat ring inside the table area.
// The widths are half a seat plus 0.5rem, so side seats don't touch the screen edge.
// Short screens, like a phone on its side, get the small seats.
const seatSize =
    "[--seat-hw:3.5rem] [--seat-hh:2.75rem] sm:[--seat-hw:4.5rem] sm:[--seat-hh:3.4rem] lg:[--seat-hw:5.5rem] lg:[--seat-hh:4.4rem] phone:[--seat-hh:3.75rem] short:[--seat-hw:5.1rem] short:[--seat-hh:1.4rem]";

// actStyle colors a player's last action by what it was
function actStyle(act: string) {
    if (act === "fold") return "bg-zinc-600 text-zinc-300";
    if (act === "all in") return "bg-rose-600 text-white";
    if (act.startsWith("bet") || act.startsWith("raise")) return "bg-orange-400 text-zinc-900";
    if (act.startsWith("call")) return "bg-emerald-500 text-zinc-900";
    return "bg-sky-400 text-zinc-900"; // check
}

// Markers are the dealer button, blind labels and bet in front of a seat, with
// the last action under them, or above them for seats below the table's middle.
// On a phone the seat's plate carries the button and blind, so just the bet and
// action go here.
function Markers({ view, player, above, badges }: { view: View; player: PlayerView; above: boolean; badges: boolean }) {
    const dealer = badges && view.running && view.button === player.seat;
    const label = badges && view.running ? blind(view, player.seat) : "";
    // the player to act is choosing again, and the bet chip shows any amount
    const turn = view.betting && view.toAct === player.seat;
    const act = turn ? "" : (view.acts[player.seat] ?? "");
    const shown = player.bet ? act.replace(/ (to )?\d+$/, "") : act;
    if (!dealer && !label && !act && !player.bet) return null;
    return (
        <div className="relative flex items-center gap-1" data-markers>
            {dealer && (
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white text-sm font-bold text-purple-800 lg:h-7 lg:w-7 lg:text-base">
                    D
                </span>
            )}
            {label && (
                <span className="flex h-6 items-center rounded-full bg-zinc-300 px-1.5 text-xs font-bold text-zinc-800">
                    {label}
                </span>
            )}
            {player.bet !== 0 && (
                <span className="flex h-6 min-w-10 items-center justify-center rounded-full bg-amber-300 px-2 text-sm font-semibold text-zinc-900 lg:h-8 lg:text-lg">
                    {player.bet}
                </span>
            )}
            {/* off the row, so the row stays put */}
            {shown && (
                <span
                    key={act}
                    className={classNames(
                        above ? "bottom-full mb-1" : "top-full mt-1",
                        "absolute left-1/2 flex h-6 -translate-x-1/2 animate-pop items-center rounded-full px-2 text-xs font-bold whitespace-nowrap shadow-md lg:text-sm",
                        actStyle(act)
                    )}
                    data-act
                >
                    {shown}
                </span>
            )}
        </div>
    );
}

export default function Table() {
    const { appState } = useContext(AppContext);
    const onPhone = useMedia(phone);
    const onTall = useMedia(tall);
    const view = appState.view;
    if (!view) return null;

    const seats = view.settings.seats;
    const bySeat = new Map(view.players.map((p) => [p.seat, p]));
    const ids = Array.from({ length: seats }, (_, i) => i + 1);
    const point = (id: number) =>
        onTall ? tallPoint(id, seats, view.you || 1) : ringPoint(seatAngle(id, seats, view.you || 1));

    return (
        <div className={"absolute inset-0 " + seatSize}>
            <div className="absolute top-1/2 left-1/2 h-[calc(0.8*(100%-2*var(--seat-hh)))] w-[calc(0.84*(100%-2*var(--seat-hw)))] -translate-1/2 rounded-full border-8 border-teal-900 bg-emerald-700" />
            {/* on a tall table, the board takes the band across the middle, between the side seats */}
            <div
                className={classNames(
                    "pointer-events-none absolute top-1/2 left-1/2 z-30 flex -translate-1/2 flex-col items-center gap-2 short:top-[53%]",
                    onTall ? "w-full px-2" : "w-[calc(0.66*(100%-2*var(--seat-hw)))] portrait:top-[40%]"
                )}
            >
                <Pot />
                <CommunityCards />
                <TableHint />
                <BuyRequests />
            </div>
            {!onPhone &&
                view.players.map((p) => (
                    <div
                        key={p.seat}
                        className="absolute z-10 -translate-1/2"
                        style={markerAt(point(p.seat))}
                    >
                        {/* the action goes on the side away from the seat */}
                        <Markers view={view} player={p} above={point(p.seat).y > 0.1} badges />
                    </div>
                ))}
            {ids.map((id) => (
                <div
                    key={id}
                    // your seat goes on top, so its menu opens over the seats beside it
                    className={classNames(
                        "absolute -translate-1/2",
                        id === view.you ? "z-40" : "z-20"
                    )}
                    style={at(point(id))}
                    data-seat={id}
                >
                    <Seat id={id} player={bySeat.get(id) ?? null} />
                    {onPhone && bySeat.has(id) && (
                        <div className="absolute -translate-1/2" style={besideSeat(point(id))}>
                            <Markers view={view} player={bySeat.get(id)!} above={point(id).y > 0.1} badges={false} />
                        </div>
                    )}
                </div>
            ))}
        </div>
    );
}
