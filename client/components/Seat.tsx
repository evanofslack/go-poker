import { useState, useContext, useEffect } from "react";
import { AppContext } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Player } from "../interfaces/index";

type seatProps = {
    player: Player | null;
    id: number;
};

export default function Seat({ player, id }: seatProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    const handleClick = () => {
        if (socket && appState.username) {
            takeSeat(socket, appState.username, id, 1000);
        }
    };

    if (player) {
        return <div className="m-2 bg-red-900 p-2 text-white">Taken by {player.username}</div>;
    } else {
        return (
            <button className="m-2  bg-gray-800 p-2 text-white" onClick={handleClick}>
                Open
            </button>
        );
    }
}
