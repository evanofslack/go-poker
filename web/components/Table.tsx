import classNames from "classnames";
import { useContext, useEffect, useState } from "react";
import { AppContext } from "../providers/AppStore";
import { PlayerView, View } from "../interfaces";
import { at, besideSeat, markerAt, ringPoint, seatAngle, tallPoint } from "../lib/seats";
import { blind } from "../lib/hands";
import { phone, tall, useMedia } from "../hooks/useMedia";
import Seat from "./Seat";
import Pot from "./Pot";
import Chips from "./Chips";
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
    if (act === "fold") return "bg-line text-fg";
    if (act === "all in") return "bg-act-all-in text-on-felt";
    if (act.startsWith("bet") || act.startsWith("raise")) return "bg-act-bet text-ink";
    if (act.startsWith("call")) return "bg-act-call text-ink";
    return "bg-act-check text-ink"; // check
}

// Markers are the blind label and bet in front of a seat, with the last action
// under them, or above them for seats below the table's middle. On a phone the
// seat's plate carries the blind, so just the bet and action go here. On the felt
// the action lines up with the row's start, clear of the dealer button outside it.
type markersProps = { view: View; player: PlayerView; above: boolean; badges: boolean; align?: "left" | "right" };

function Markers({ view, player, above, badges, align }: markersProps) {
    const label = badges && view.running ? blind(view, player.seat) : "";
    // the player to act is choosing again, and the bet chip shows any amount
    const turn = view.betting && view.toAct === player.seat;
    const act = turn ? "" : (view.acts[player.seat] ?? "");
    const shown = player.bet ? act.replace(/ (to )?\d+$/, "") : act;
    if (!label && !act && !player.bet) return null;
    return (
        <div className="relative flex items-center gap-1" data-markers>
            {label && (
                <span className="puck flex h-6 items-center rounded-full px-1.5 text-xs font-bold text-ink [--puck:var(--color-blind)]">
                    {label}
                </span>
            )}
            {player.bet !== 0 && (
                <span className="flex items-center gap-1">
                    <Chips amount={player.bet} bb={view.settings.bb} className="size-5 lg:size-6" />
                    <span className="flex h-6 items-center rounded-full bg-felt-dark/80 px-1.5 text-sm font-semibold text-on-felt lg:h-7 lg:text-lg">
                        {player.bet}
                    </span>
                </span>
            )}
            {/* off the row, so the row stays put */}
            {shown && (
                <span
                    key={act}
                    className={classNames(
                        above ? "bottom-full mb-1" : "top-full mt-1",
                        align === "left" ? "left-0" : align === "right" ? "right-0" : "left-1/2 -translate-x-1/2",
                        "absolute flex h-6 animate-pop items-center rounded-full px-2 text-xs font-bold whitespace-nowrap shadow-md lg:text-sm",
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
    // animate only once the table has settled after a join or rejoin, so the
    // backlog doesn't play out as if it just happened
    const [motion, setMotion] = useState(false);
    useEffect(() => {
        setMotion(false);
        const t = setTimeout(() => setMotion(true), 500);
        return () => clearTimeout(t);
    }, [appState.welcomed]);
    const view = appState.view;
    if (!view) return null;

    const seats = view.settings.seats;
    const bySeat = new Map(view.players.map((p) => [p.seat, p]));
    const ids = Array.from({ length: seats }, (_, i) => i + 1);
    const point = (id: number) =>
        onTall ? tallPoint(id, seats, view.you || 1) : ringPoint(seatAngle(id, seats, view.you || 1));
    // your own cards are bigger, so your markers sit further in to clear them
    const markerPoint = (id: number) => {
        const p = point(id);
        return id === view.you ? { x: p.x * 0.75, y: p.y * 0.75 } : p;
    };
    // a seat's markers start at its marker point and run toward the middle of the
    // table, and the dealer button sits just outside them, toward the seat
    const right = (id: number) => point(id).x > 0.05;
    const dealer = !onPhone && view.running && bySeat.has(view.button);

    return (
        <div className={"absolute inset-0 " + seatSize} data-motion={motion ? appState.prefs.motion : undefined}>
            <div className="absolute top-1/2 left-1/2 h-[calc(0.8*(100%-2*var(--seat-hh)))] w-[calc(0.84*(100%-2*var(--seat-hw)))] -translate-1/2 rounded-full felt" />
            {/* on a tall table, the board takes the band across the middle, between the side seats */}
            <div
                className={classNames(
                    "pointer-events-none absolute top-1/2 left-1/2 z-30 flex -translate-1/2 flex-col items-center gap-2 short:top-[53%]",
                    onTall ? "w-full px-2" : "w-[calc(0.66*(100%-2*var(--seat-hw)))] portrait:top-[40%]"
                )}
                // each extra run of the board lifts the column, so the runs clear the markers below it
                style={!onTall && view.runs > 1 ? { marginTop: `${-(view.runs - 1) * 1.75}rem` } : undefined}
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
                        className={classNames(
                            "absolute z-10 -translate-y-1/2",
                            right(p.seat) ? "-translate-x-[calc(100%+0.25rem)]" : "translate-x-1"
                        )}
                        style={markerAt(markerPoint(p.seat))}
                    >
                        {/* the action goes on the side away from the seat */}
                        <Markers view={view} player={p} above={point(p.seat).y > 0.1} badges align={right(p.seat) ? "right" : "left"} />
                    </div>
                ))}
            {/* one button for the table, so it glides to the next seat each hand */}
            {dealer && (
                <div
                    className={classNames(
                        "absolute z-10 -translate-y-1/2 motion:transition-[left,top,translate] motion:duration-500 motion:ease-out",
                        right(view.button) ? "translate-x-1" : "-translate-x-[calc(100%+0.25rem)]"
                    )}
                    style={markerAt(markerPoint(view.button))}
                    data-markers
                    data-dealer
                >
                    <span className="puck flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold text-ink lg:h-7 lg:w-7 lg:text-base">
                        D
                    </span>
                </div>
            )}
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
