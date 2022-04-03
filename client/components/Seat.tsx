import { useState, useContext, useEffect } from "react";
import { AppContext, AppStoreProvider } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import classNames from "classnames";

type seatProps = {
    player: Player | null;
    id: number;
};

const getActive = (player: Player, game: Game) => {
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

        "px-4 py-4 rounded-xl m-4 h-36 w-36"
    );
};

export default function Seat({ player, id }: seatProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const handleClick = () => {
        if (socket && appState.username) {
            takeSeat(socket, appState.username, id, 1000);
        }
    };

    if (player && appState.game) {
        let cards = ["?", "?"];
        if (appState.clientID == player?.id) {
            cards = player.cards;
        }
        // This is the player's seat
        return (
            <div className="">
                <div className={getActive(player, appState.game)}>
                    <p className="text-3xl font-semibold">{player.username}</p>
                    <p>{player.stack}</p>
                    <p>{cards}</p>
                    {player.bet !== 0 && (
                        <p className="flex h-8 w-8 items-center justify-center rounded-3xl bg-yellow-400">
                            {player.bet}
                        </p>
                    )}
                </div>
            </div>
        );
        // player already sat down, and this seat does not belong to them
    } else if (player?.id != appState.clientID) {
        return (
            <button className="m-4 h-36 w-36 rounded-2xl bg-green-800 p-2 text-white opacity-30">
                <h2 className="text-4xl">{id}</h2>
            </button>
        );
        // player has not yet sat down, all seats are open
    } else {
        return (
            <button
                className="m-4 h-36 w-36 rounded-2xl bg-gray-800 p-2 text-white"
                onClick={handleClick}
            >
                <h2 className="text-4xl">{id}</h2>
                <p className="opacity-70">Open</p>
            </button>
        );
    }
}
