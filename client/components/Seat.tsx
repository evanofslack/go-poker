import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { Game, Player } from "../interfaces/index";
import Card from "./Card";
import BuyIn from "./BuyIn";
import classNames from "classnames";

type seatProps = {
    player: Player | null;
    id: number;
    reveal: boolean;
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
    const winner = player.position == game?.pots[game.pots.length - 1]?.winningPlayerNums[0];
    return classNames(
        {
            // betting and player's turn
            "shadow-[0px_0px_40px_2px_rgba(255,255,255,255.3)] bg-neutral-100 text-zinc-900":
                action && game.betting,

            // betting and not player's turn
            "bg-zinc-900 text-neutral-100": !action && game.betting,

            // betting over and winner
            "shadow-[0px_0px_60px_20px_rgba(100,98,92,255.3)] bg-amber-200 text-zinc-900":
                winner && !game.betting,

            // betting over and not winner
            "bg-zinc-900 text-neutral-100 ": !winner && !game.betting,
        },

        "rounded-xl m-4 h-20 w-56 flex flex-row justify-start items-center z-2"
    );
}

export default function Seat({ player, id, reveal }: seatProps) {
    const { appState, dispatch } = useContext(AppContext);
    const [sitDown, setSitDown] = useState(false);

    let hidden = false;
    if (player && appState.game) {
        if (appState.game.running) {
            if (appState.clientID !== player?.uuid) {
                hidden = true;
            }
        }
        // This is the player's seat
        return (
            <div className="relative">
                <div className={active(player, appState.game)}>
                    <div className="relative right-2 flex flex-row items-center justify-center">
                        {player.cards.map((c, i) => (
                            <div key={i} className="mx-0.5">
                                <Card
                                    card={c}
                                    placeholder={false}
                                    folded={!player.in}
                                    hidden={reveal ? false : hidden}
                                />
                            </div>
                        ))}
                    </div>
                    <div className="flex flex-col py-4 pr-2 pl-1">
                        <p className="-mb-1 text-lg font-normal">{player.username}</p>
                        <p className="text-lg font-semibold">{player.stack}</p>
                    </div>
                </div>
                <div className={chipPosition(id)}>
                    {appState.game.running && appState.game.dealer == player.position && (
                        <div className="mx-3 my-3 flex h-7 w-8 items-center justify-center rounded-[50%] bg-white text-xl font-bold text-purple-800">
                            D
                        </div>
                    )}
                    {player.bet !== 0 && (
                        <p className=" flex h-8 w-12 items-center justify-center rounded-3xl bg-amber-300 text-xl font-semibold text-zinc-900">
                            {player.bet}
                        </p>
                    )}
                </div>
            </div>
        );
        // player already sat down, and this seat does not belong to them
    } else if (player?.uuid != appState.clientID && !appState.game?.running) {
        return (
            <div>
                <button className="m-4 h-20 w-56 rounded-2xl bg-neutral-700 p-2 text-neutral-400 opacity-20">
                    <h2 className="text-4xl">{id}</h2>
                </button>
            </div>
        );
        // player has not yet sat down, all seats are open
    } else if (!appState.game?.running) {
        return (
            <div>
                {sitDown && (
                    <div className="m-4 h-20 w-56 rounded-2xl bg-neutral-700 text-neutral-100">
                        <BuyIn seatID={id} sitDown={sitDown} setSitDown={setSitDown} />
                    </div>
                )}
                {!sitDown && (
                    <button
                        className="m-4 h-20 w-56 rounded-2xl bg-neutral-700 p-2 text-neutral-100"
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
