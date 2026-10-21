import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { removeBot, takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import Card, { cardSize } from "./Card";
import BuyIn from "./BuyIn";
import Dialog from "./Dialog";
import AddBot from "./AddBot";
import Countdown from "./Countdown";
import SeatMenu from "./SeatMenu";
import { isWinner, winningCards } from "../lib/hands";
import classNames from "classnames";

const plate =
    "w-24 rounded-xl px-2 py-1 text-center sm:w-32 lg:w-40 lg:py-1.5 short:w-24 short:py-0.5";

function plateStyle(player: Player, game: Game) {
    const action = game.betting && player.position === game.action;
    const counting = action && game.turnEndsIn > 0;
    const winner = !game.betting && isWinner(game, player.position);
    return classNames(plate, "relative", {
        "bg-zinc-100 text-zinc-900 shadow-[0_0_40px_2px_rgba(255,255,255,0.3)]": action,
        "bg-amber-200 text-zinc-900 shadow-[0_0_60px_20px_rgba(100,98,92,0.3)]": winner,
        "bg-panel text-fg": !action && !winner,
        // sitting out or out of chips
        "opacity-50": !player.ready && !action && !winner,
        // clear the cards that overlap the top of the plate
        "pt-2.5 short:py-0.5 short:pl-3": true,
        // room for the countdown along the bottom
        "pb-2.5 short:pb-2 lg:pb-3": counting,
    });
}

type playerSeatProps = {
    player: Player;
    game: Game;
    own: boolean;
    bot?: string; // the bot's level, if it's a bot
    host: boolean; // the viewer is the host
};

function PlayerSeat({ player, game, own, bot, host }: playerSeatProps) {
    const socket = useSocket();
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
                <p
                    className={classNames(
                        "truncate text-sm leading-tight sm:text-base short:text-xs",
                        bot && "flex items-center justify-center"
                    )}
                >
                    {/* a bot's long name gives way to its badge */}
                    {bot ? <span className="truncate">{player.username}</span> : player.username}
                    {bot && (
                        // just "bot" on small plates, so it isn't cut off
                        <span
                            className="ml-1 shrink-0 rounded-full bg-badge px-1.5 text-xs text-fg"
                            title={`${bot} bot`}
                            data-bot-badge
                        >
                            <span className="hidden sm:inline">{bot} </span>bot
                        </span>
                    )}
                </p>
                <p className="text-sm leading-tight font-semibold sm:text-lg short:text-sm">
                    {player.stack}
                </p>
                {game.betting && game.action === player.position && (
                    <span className="sr-only">to act</span>
                )}
                {!player.ready && (
                    <span className="absolute -top-2.5 left-1/2 z-20 -translate-x-1/2 rounded-full bg-badge px-2 text-xs whitespace-nowrap text-fg">
                        {player.stack > 0 ? "sitting out" : "out of chips"}
                    </span>
                )}
                {own && <SeatMenu player={player} game={game} />}
                {bot && host && socket && !game.ended && (
                    // on the corner, clear of the name and badge
                    <button
                        className="absolute -right-2 -bottom-2 z-20 flex h-5 w-5 items-center justify-center rounded-full bg-raised text-sm leading-none text-muted hover:text-fg"
                        title="remove bot"
                        aria-label={`remove ${player.username}`}
                        onClick={() => removeBot(socket, player.seatID)}
                    >
                        ×
                    </button>
                )}
                {game.betting && game.action === player.position && game.turnEndsIn > 0 && (
                    <Countdown endsIn={game.turnEndsIn} total={game.settings.turnTime} />
                )}
            </div>
        </div>
    );
}

// OpenSeat lets someone without a seat sit down, after choosing a buy-in. The
// host can sit a bot there instead.
function OpenSeat({ id, game, host }: { id: number; game: Game; host: boolean }) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
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
                        min={game.settings.minBuy}
                        max={game.settings.maxBuy}
                        initial={100 * game.settings.bb}
                        label="sit down"
                        onSubmit={(amount) => {
                            if (socket && appState.username) {
                                takeSeat(socket, appState.username, id, amount);
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
                    <AddBot seat={id} game={game} onDone={close} />
                </Dialog>
            )}
        </>
    );
}

// BotSeat is an empty seat as the seated host sees it, for adding a bot
function BotSeat({ id, game }: { id: number; game: Game }) {
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
                    <AddBot seat={id} game={game} onDone={() => setOpen(false)} />
                </Dialog>
            )}
        </>
    );
}

export default function Seat({ player, id }: { player: Player | null; id: number }) {
    const { appState } = useContext(AppContext);
    const game = appState.game;
    if (!game) return null;
    const host = game.host === appState.username;

    if (player) {
        // the server only sends cards this client may see
        return (
            <PlayerSeat
                player={player}
                game={game}
                own={player.uuid === appState.clientID}
                bot={game.bots[player.uuid]}
                host={host}
            />
        );
    }
    // nobody sits down, and no bots join, once the session has ended
    if (game.ended) return null;
    const seated = game.players.some((p) => p.uuid === appState.clientID);
    if (!seated) {
        return <OpenSeat id={id} game={game} host={host} />;
    }
    if (host) {
        return <BotSeat id={id} game={game} />;
    }
    // seated players see empty seats faintly until the game starts
    if (game.running) return null;
    return (
        <div className={classNames(plate, "bg-raised py-2 text-lg text-muted opacity-20")}>
            {id}
        </div>
    );
}
