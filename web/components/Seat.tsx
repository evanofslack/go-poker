import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import Card from "./Card";
import BuyIn from "./BuyIn";
import Countdown from "./Countdown";
import { isWinner, winningCards } from "../lib/hands";
import classNames from "classnames";

type seatProps = {
    player: Player | null;
    id: number;
};

function chipPosition(id: number) {
    return classNames(
        {
            "right-[60%] top-[-45%] flex-row": id === 1,
            "right-[30%] top-[-40%] flex-row": id === 2,
            "right-[-20%] top-[20%] flex-col": id === 3,
            "right-[30%] bottom-[-40%] flex-row": id === 4,
            "right-[60%] bottom-[-40%] flex-row": id === 5,
            "left-[-23%] top-[15%] flex-col": id === 6,
        },
        "absolute flex items-center justify-start z-10"
    );
}

function active(player: Player, game: Game) {
    const action = player.position === game.action;
    const winner = isWinner(game, player.position);
    return classNames(
        {
            // betting and player's turn
            "shadow-[0px_0px_40px_2px_rgba(255,255,255,255.3)] bg-zinc-100 text-zinc-900":
                action && game.betting,

            // betting and not player's turn
            "bg-panel text-fg": !action && game.betting,

            // betting over and winner
            "shadow-[0px_0px_60px_20px_rgba(100,98,92,255.3)] bg-amber-200 text-zinc-900":
                winner && !game.betting,

            // betting over and not winner
            "bg-panel text-fg ": !winner && !game.betting,
        },

        "relative rounded-xl m-4 h-20 w-56 flex flex-row justify-start items-center z-2"
    );
}

// blind labels the small and big blind seats for the current hand
function blind(game: Game, position: number) {
    if (game.stage === 1) return "";
    if (game.sb === position) return "SB";
    if (game.bb === position) return "BB";
    return "";
}

export default function Seat({ player, id }: seatProps) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [sitDown, setSitDown] = useState(false);

    const seated = appState.game?.players.some((p) => p.uuid === appState.clientID);
    if (player && appState.game) {
        // This is the player's seat. The server only sends cards this client may see.
        return (
            <div className="relative">
                <div className={active(player, appState.game)}>
                    <div className="relative right-2 flex flex-row items-center justify-center">
                        {player.cards.map((c, i) => (
                            <div key={i} className="mx-0.5">
                                <Card
                                    card={c}
                                    placeholder={false}
                                    // everyone is out between hands; cards shown then reached showdown
                                    folded={!player.in && !(appState.game!.stage === 1 && c)}
                                    hidden={!c}
                                    highlight={winningCards(appState.game!).has(String(c))}
                                />
                            </div>
                        ))}
                    </div>
                    <div className="flex flex-col py-4 pr-2 pl-1">
                        <p className="-mb-1 text-lg font-normal">{player.username}</p>
                        <p className="text-lg font-semibold">{player.stack}</p>
                    </div>
                    {appState.game.betting &&
                        appState.game.action === player.position &&
                        appState.game.turnEndsIn > 0 && (
                            <Countdown endsIn={appState.game.turnEndsIn} />
                        )}
                </div>
                <div className={chipPosition(id)}>
                    {appState.game.running && appState.game.dealer == player.position && (
                        <div className="mx-3 my-3 flex h-7 w-8 items-center justify-center rounded-full bg-white text-xl font-bold text-purple-800">
                            D
                        </div>
                    )}
                    {appState.game.running && blind(appState.game, player.position) && (
                        <div className="mr-3 flex h-7 items-center justify-center rounded-full bg-zinc-300 px-2 text-sm font-bold text-zinc-800">
                            {blind(appState.game, player.position)}
                        </div>
                    )}
                    {player.bet !== 0 && (
                        <p className=" flex h-8 w-12 items-center justify-center rounded-full bg-amber-300 text-xl font-semibold text-zinc-900">
                            {player.bet}
                        </p>
                    )}
                </div>
            </div>
        );
        // player already sat down, and this seat does not belong to them
    } else if (seated && !appState.game?.running) {
        return (
            <div>
                <button className="m-4 h-20 w-56 rounded-xl bg-raised p-2 text-muted opacity-20">
                    <h2 className="text-4xl">{id}</h2>
                </button>
            </div>
        );
        // player has not yet sat down, all seats are open
    } else if (!appState.game?.running) {
        return (
            <div>
                {sitDown && appState.game && (
                    <div className="m-4 h-20 w-56 rounded-xl bg-raised text-fg">
                        <BuyIn
                            max={appState.game.config.maxBuy}
                            initial={100 * appState.game.config.bb}
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
                )}
                {!sitDown && (
                    <button
                        className="m-4 h-20 w-56 rounded-xl bg-raised p-2 text-fg"
                        onClick={() => setSitDown(!sitDown)}
                    >
                        <h2 className="text-4xl">{id}</h2>
                        <p className="opacity-70">Open</p>
                    </button>
                )}
            </div>
        );
    } else {
        return <div></div>;
    }
}
