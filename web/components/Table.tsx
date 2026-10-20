import Seat from "./Seat";
import Felt from "./Felt";
import { Game as GameType, Player } from "../interfaces";
import { AppContext } from "../providers/AppStore";
import { sendLog, dealGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import React, { useContext, useState, useEffect } from "react";

type tableProps = {
    players: (Player | null)[];
    setPlayers: React.Dispatch<React.SetStateAction<(Player | null)[]>>;
};

function getWinner(game: GameType) {
    const winnerNum = game.pots[game.pots.length - 1].winningPlayerNums[0];
    const winningPlayer = game.players.filter((player) => player.position == winnerNum)[0];
    return winningPlayer;
}

function handleWinner(game: GameType | null, socket: WebSocket | null) {
    if (!game || !socket) {
        return null;
    }
    if (game && game.stage === 1 && game.pots.length !== 0) {
        const winningPlayer = getWinner(game);
        const pot = game.pots[game.pots.length - 1].amount;
        const message = winningPlayer.username + " wins " + pot;
        sendLog(socket, message);
    }
}

export default function Table({ players, setPlayers }: tableProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;

    // map game players to their visual seats
    useEffect(() => {
        let updatedPlayers: (Player | null)[] = [...players];
        if (game?.players == null) {
            return;
        }
        for (let i = 0; i < game.players.length; i++) {
            updatedPlayers[game.players[i].seatID - 1] = game.players[i];
        }
        setPlayers(updatedPlayers);
    }, [game?.players]);

    useEffect(() => {
        // this effect triggers when betting is over
        if (game && game.stage === 1 && game.pots.length !== 0) {
            handleWinner(game, socket);
            const timer = setTimeout(() => {
                if (socket) {
                    dealGame(socket);
                }
            }, 5000);
            return () => {
                clearTimeout(timer);
            };
        }
    }, [game?.pots]);

    return (
        <div className="mt-2 grid h-3/4 w-full max-w-(--breakpoint-xl) grid-cols-5 grid-rows-5 sm:mt-28 sm:h-3/5">
            <div></div>
            <Seat
                key={3}
                player={players[3]}
                id={4}
            />
            <div></div>
            <Seat
                key={4}
                player={players[4]}
                id={5}
            />
            <div></div>
            <div></div>
            <div className="col-span-3 row-span-3 flex items-center justify-center">
                <Felt />
            </div>
            <div></div>
            <Seat
                key={2}
                player={players[2]}
                id={3}
            />
            <Seat
                key={5}
                player={players[5]}
                id={6}
            />
            <div></div>
            <div></div>
            <div></div>
            <Seat
                key={1}
                player={players[1]}
                id={2}
            />
            <div></div>
            <Seat
                key={0}
                player={players[0]}
                id={1}
            />
            <div></div>
        </div>
    );
}
