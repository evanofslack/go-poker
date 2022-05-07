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
        console.log("getting winner");
        const winningPlayer = getWinner(game);
        const pot = game.pots[game.pots.length - 1].amount;
        const message = winningPlayer.username + " wins " + pot;
        sendLog(socket, message);
    }
}

function getRevealedPlayers(game: GameType) {
    const revealedNums = game.pots[game.pots.length - 1].eligiblePlayerNums;
    // if only one player was eligible for the pot (everyone else folded), then they do not have to reveal
    if (revealedNums.length <= 1) {
        return [];
    }
    const revealedPlayers = game.players.filter((player) => revealedNums.includes(player.position));
    return revealedPlayers;
}

export default function Table({ players, setPlayers }: tableProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;
    const [revealedPlayers, setRevealedPlayers] = useState<Player[]>([]);

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
            setRevealedPlayers(getRevealedPlayers(game));
            handleWinner(game, socket);
            const timer = setTimeout(() => {
                setRevealedPlayers([]);
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
        <div className="mt-2 grid h-3/4 w-full max-w-screen-xl grid-cols-5 grid-rows-5 sm:mt-28 sm:h-3/5">
            <div></div>
            <Seat
                key={3}
                player={players[3]}
                id={4}
                reveal={players[3] ? revealedPlayers.includes(players[3]) : false}
            />
            <div></div>
            <Seat
                key={4}
                player={players[4]}
                id={5}
                reveal={players[4] ? revealedPlayers.includes(players[4]) : false}
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
                reveal={players[2] ? revealedPlayers.includes(players[2]) : false}
            />
            <Seat
                key={5}
                player={players[5]}
                id={6}
                reveal={players[5] ? revealedPlayers.includes(players[5]) : false}
            />
            <div></div>
            <div></div>
            <div></div>
            <Seat
                key={1}
                player={players[1]}
                id={2}
                reveal={players[1] ? revealedPlayers.includes(players[1]) : false}
            />
            <div></div>
            <Seat
                key={0}
                player={players[0]}
                id={1}
                reveal={players[0] ? revealedPlayers.includes(players[0]) : false}
            />
            <div></div>
        </div>
    );
}
