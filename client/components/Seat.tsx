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

const getActive = () => classNames("bg-red-900", "m-2", "p-2", "text-white");

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
        return (
            <div>
                <div className={getActive()}>
                    <p>player: {player.username}</p>
                    <p>stack: {player.stack}</p>
                    <p>cards: {cards}</p>
                </div>
            </div>
        );
    } else {
        return (
            <button className="m-2  bg-gray-800 p-2 text-white" onClick={handleClick}>
                Open
            </button>
        );
    }
}
