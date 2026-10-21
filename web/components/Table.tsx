import classNames from "classnames";
import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { Game, Player } from "../interfaces";
import { at, seatAngle } from "../lib/seats";
import Seat from "./Seat";
import Pot from "./Pot";
import CommunityCards from "./CommunityCards";
import TableHint from "./TableHint";

// half a seat's size at each breakpoint, which keeps the seat ring inside the table area.
// Short screens, like a phone on its side, get the small seats.
const seatSize =
    "[--seat-hw:3rem] [--seat-hh:2.75rem] sm:[--seat-hw:4rem] sm:[--seat-hh:3.4rem] lg:[--seat-hw:5rem] lg:[--seat-hh:4.4rem] short:[--seat-hw:4.6rem] short:[--seat-hh:1.4rem]";

// blind labels the small and big blind seats for the current hand
function blind(game: Game, position: number) {
    if (game.stage === 1) return "";
    if (game.sb === position) return "SB";
    if (game.bb === position) return "BB";
    return "";
}

// Markers are the dealer button, blind labels, and bet in front of a seat
function Markers({ game, player }: { game: Game; player: Player }) {
    const dealer = game.running && game.dealer === player.position;
    const label = game.running ? blind(game, player.position) : "";
    if (!dealer && !label && !player.bet) return null;
    return (
        <div className="flex items-center gap-1">
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
        </div>
    );
}

export default function Table() {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game) return null;

    const seats = game.seats || 8;
    const own = game.players.find((p) => p.uuid === appState.clientID);
    const bySeat = new Map(game.players.map((p) => [p.seatID, p]));
    const ids = Array.from({ length: seats }, (_, i) => i + 1);
    const angle = (id: number) => seatAngle(id, seats, own?.seatID ?? 1);

    return (
        <div className={"absolute inset-0 " + seatSize}>
            <div className="absolute top-1/2 left-1/2 h-[calc(0.8*(100%-2*var(--seat-hh)))] w-[calc(0.84*(100%-2*var(--seat-hw)))] -translate-1/2 rounded-full border-8 border-teal-900 bg-emerald-700" />
            <div className="pointer-events-none absolute top-1/2 left-1/2 z-30 flex w-[calc(0.66*(100%-2*var(--seat-hw)))] -translate-1/2 flex-col items-center gap-2 portrait:top-[40%] short:top-[53%]">
                <Pot />
                <CommunityCards />
                <TableHint />
            </div>
            {game.players.map((p) => (
                <div
                    key={p.uuid}
                    className="absolute z-10 -translate-1/2"
                    style={at(angle(p.seatID), 0.6)}
                >
                    <Markers game={game} player={p} />
                </div>
            ))}
            {ids.map((id) => (
                <div
                    key={id}
                    // your seat goes on top, so its menu opens over the seats beside it
                    className={classNames(
                        "absolute -translate-1/2",
                        id === own?.seatID ? "z-40" : "z-20"
                    )}
                    style={at(angle(id))}
                >
                    <Seat id={id} player={bySeat.get(id) ?? null} />
                </div>
            ))}
        </div>
    );
}
