import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import Card from "./Card";
import classNames from "classnames";

type seatProps = {
    player: Player | null;
    id: number;
};

function seatPosition(id: number) {
    return classNames(
        {
            "right-[5%] bottom-[-23%]": id === 1,
            "left-[5%] bottom-[-23%]": id === 2,
            "left-[-23%] top-[35%]": id === 3,
            "left-[5%] top-[-23%]": id === 4,
            "right-[5%] top-[-23%]": id === 5,
            "right-[-23%] top-[35%]": id === 6,
        },
        "absolute"
    );
}

function chipPosition(id: number) {
    return classNames(
        {
            "right-[20%] bottom-[-30%]": id === 1,
            "left-[20%] bottom-[-30%]": id === 2,
            "left-[-15%] top-[20%]": id === 3,
            "left-[20%] top-[-30%]": id === 4,
            "right-[60%] bottom-[-40%] flex-row": id === 5,
            "left-[-23%] top-[15%] flex-col": id === 6,
        },
        "absolute flex items-center justify-start"
    );
}

function active(player: Player, game: Game) {
    const action = player.seatID === game.action;
    const winner = player.seatID == game?.pots[game.pots.length - 1]?.winningPlayerNums[0];
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

        "rounded-xl m-4 h-20 w-56 flex flex-row justify-start items-center"
    );
}

export default function Seat({ player, id }: seatProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const handleClick = () => {
        if (socket && appState.username) {
            takeSeat(socket, appState.username, id, 1000);
        }
    };

    if (player && appState.game) {
        let cards = player.cards;
        if (appState.game.running) {
            if (appState.clientID !== player?.id) {
                cards = ["?", "?"];
            }
        }
        // This is the player's seat
        return (
            <div className={seatPosition(id)}>
                <div className={active(player, appState.game)}>
                    <div className="relative right-2 flex flex-row items-center justify-center">
                        {cards.map((c, i) => (
                            <div key={i} className="mx-0.5">
                                <Card card={c} width={80} height={110} />
                            </div>
                        ))}
                    </div>
                    <div className="flex flex-col py-4 pr-2 pl-1">
                        <p className="-mb-1 text-lg font-normal">{player.username}</p>
                        <p className="text-lg font-semibold">{player.stack}</p>
                    </div>
                </div>
                <div className={chipPosition(id)}>
                    {appState.game.running && appState.game.dealer == player.seatID && (
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
    } else if (player?.id != appState.clientID) {
        return (
            <div className={seatPosition(id)}>
                <button className="m-4 h-20 w-56 rounded-2xl bg-neutral-700 p-2 text-neutral-400 opacity-20">
                    <h2 className="text-4xl">{id}</h2>
                </button>
            </div>
        );
        // player has not yet sat down, all seats are open
    } else {
        return (
            <div className={seatPosition(id)}>
                <button
                    className="m-4 h-20 w-56 rounded-2xl bg-neutral-700 p-2 text-neutral-100"
                    onClick={handleClick}
                >
                    <h2 className="text-4xl">{id}</h2>
                    <p className="opacity-70">Open</p>
                </button>
            </div>
        );
    }
}
