import { useState, useContext } from "react";
import { createPortal } from "react-dom";
import { AppContext } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import Card, { cardSize } from "./Card";
import BuyIn from "./BuyIn";
import Countdown from "./Countdown";
import SeatMenu from "./SeatMenu";
import { isWinner, winningCards } from "../lib/hands";
import classNames from "classnames";

const plate =
    "w-24 rounded-xl px-2 py-1 text-center sm:w-32 lg:w-40 lg:py-1.5 short:w-24 short:py-0.5";

function plateStyle(player: Player, game: Game) {
    const action = game.betting && player.position === game.action;
    const winner = !game.betting && isWinner(game, player.position);
    return classNames(plate, "relative", {
        "bg-zinc-100 text-zinc-900 shadow-[0_0_40px_2px_rgba(255,255,255,0.3)]": action,
        "bg-amber-200 text-zinc-900 shadow-[0_0_60px_20px_rgba(100,98,92,0.3)]": winner,
        "bg-panel text-fg": !action && !winner,
        // sitting out or out of chips
        "opacity-50": !player.ready && !action && !winner,
        // clear the cards that overlap the top of the plate
        "pt-2.5 short:py-0.5 short:pl-3": true,
    });
}

function PlayerSeat({ player, game, own }: { player: Player; game: Game; own: boolean }) {
    const winning = winningCards(game);
    return (
        <div className="flex flex-col items-center short:flex-row">
            <div
                className={classNames(
                    "relative z-10 -mb-2 flex gap-0.5 short:-mr-2 short:mb-0",
                    cardSize.h
                )}
            >
                {player.cards.map((c, i) => (
                    <Card
                        key={i}
                        card={c}
                        placeholder={false}
                        // everyone is out between hands; cards shown then reached showdown
                        folded={!player.in && !(game.stage === 1 && c)}
                        hidden={!c}
                        highlight={winning.has(String(c))}
                    />
                ))}
            </div>
            <div className={plateStyle(player, game)}>
                <p className="truncate text-sm leading-tight sm:text-base short:text-xs">
                    {player.username}
                </p>
                <p className="text-sm leading-tight font-semibold sm:text-lg short:text-sm">
                    {player.stack}
                </p>
                {!player.ready && (
                    <span className="absolute -top-2.5 left-1/2 z-20 -translate-x-1/2 rounded-full bg-zinc-600 px-2 text-xs whitespace-nowrap text-fg">
                        {player.stack > 0 ? "sitting out" : "out of chips"}
                    </span>
                )}
                {own && <SeatMenu player={player} game={game} />}
                {game.betting && game.action === player.position && game.turnEndsIn > 0 && (
                    <Countdown endsIn={game.turnEndsIn} />
                )}
            </div>
        </div>
    );
}

// OpenSeat lets someone without a seat sit down, after choosing a buy-in
function OpenSeat({ id, game }: { id: number; game: Game }) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [sitDown, setSitDown] = useState(false);

    return (
        <>
            <button
                className={classNames(
                    plate,
                    "border-2 border-dashed border-zinc-600 bg-raised/60 text-fg hover:bg-raised"
                )}
                onClick={() => setSitDown(true)}
            >
                <span className="text-lg sm:text-2xl">{id}</span>
                <span className="block text-xs opacity-70 sm:text-sm">Open</span>
            </button>
            {sitDown &&
                // a portal, since the seat's transform would otherwise contain a fixed overlay
                createPortal(
                    <div
                        className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
                        onClick={(e) => e.target === e.currentTarget && setSitDown(false)}
                    >
                        <div className="w-72 rounded-xl bg-panel p-4 text-fg shadow-xl">
                            <p className="mb-2 font-semibold">sit in seat {id}</p>
                            <BuyIn
                                max={game.config.maxBuy}
                                initial={100 * game.config.bb}
                                label="sit down"
                                onSubmit={(amount) => {
                                    if (socket && appState.username) {
                                        takeSeat(socket, appState.username, id, amount);
                                    }
                                    setSitDown(false);
                                }}
                                onCancel={() => setSitDown(false)}
                            />
                        </div>
                    </div>,
                    document.body
                )}
        </>
    );
}

export default function Seat({ player, id }: { player: Player | null; id: number }) {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game) return null;

    if (player) {
        // the server only sends cards this client may see
        return <PlayerSeat player={player} game={game} own={player.uuid === appState.clientID} />;
    }
    const seated = game.players.some((p) => p.uuid === appState.clientID);
    if (!seated) {
        return <OpenSeat id={id} game={game} />;
    }
    // seated players see empty seats faintly until the game starts
    if (game.running) return null;
    return (
        <div className={classNames(plate, "bg-raised py-2 text-lg text-muted opacity-20")}>
            {id}
        </div>
    );
}
