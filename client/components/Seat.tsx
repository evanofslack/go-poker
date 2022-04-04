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

function position(id: number) {
    return classNames(
        {
            "right-[20%] bottom-[-30%]": id === 1,
            "left-[20%] bottom-[-30%]": id === 2,
            "left-[-15%] top-[20%]": id === 3,
            "left-[20%] top-[-30%]": id === 4,
            "right-[20%] top-[-30%]": id === 5,
            "right-[-15%] top-[20%]": id === 6,
        },
        "absolute"
    );
}

function active(player: Player, game: Game) {
    const action = player.seatID === game.action;
    const winner = player.seatID == game?.pots[game.pots.length - 1]?.winningPlayerNums[0];
    return classNames(
        {
            // betting and player's turn
            "shadow-[0px_0px_30px_9px_rgba(255,255,255,255.3)] bg-white text-black":
                action && game.betting,

            // betting and not player's turn
            "bg-black text-white": !action && game.betting,

            // betting over and winner
            "shadow-[0px_0px_30px_9px_rgba(255,255,255,255.3)] bg-yellow-200 text-black":
                winner && !game.betting,

            // betting over and not winner
            "bg-black text-white ": !winner && !game.betting,
        },

        "px-4 py-4 rounded-xl m-4 h-40 w-36"
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
            <div className={position(id)}>
                <div className={active(player, appState.game)}>
                    <p className="text-3xl font-semibold">{player.username}</p>
                    <p>{player.stack}</p>
                    <div className="flex flex-row">
                        {cards.map((c, i) => (
                            <div key={i} className="m-1">
                                <Card card={c} width={50} height={65} />
                            </div>
                        ))}
                    </div>
                    {player.bet !== 0 && (
                        <p className="flex h-8 w-12 items-center justify-center rounded-3xl bg-yellow-400 text-black">
                            {player.bet}
                        </p>
                    )}
                </div>
            </div>
        );
        // player already sat down, and this seat does not belong to them
    } else if (player?.id != appState.clientID) {
        return (
            <div className={position(id)}>
                <button className="m-4 h-40 w-36 rounded-2xl bg-green-800 p-2 text-white opacity-30">
                    <h2 className="text-4xl">{id}</h2>
                </button>
            </div>
        );
        // player has not yet sat down, all seats are open
    } else {
        return (
            <div className={position(id)}>
                <button
                    className="m-4 h-40 w-36 rounded-2xl bg-gray-800 p-2 text-white"
                    onClick={handleClick}
                >
                    <h2 className="text-4xl">{id}</h2>
                    <p className="opacity-70">Open</p>
                </button>
            </div>
        );
    }
}
