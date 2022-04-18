import CommunityCards from "./CommunityCards";
import Pot from "./Pot";
import Seat from "./Seat";
import { Game as GameType, Player } from "../interfaces";
import { AppContext } from "../providers/AppStore";
import { sendLog, dealGame } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import React, { useContext, useState, useEffect } from "react";

type tableProps = {
    players: (Player | null)[];
    setPlayers: React.Dispatch<React.SetStateAction<(Player | null)[]>>;
};

function dealerLog(game: GameType) {
    let user = game.players[game.dealer].username;
    return user + " is dealer";
}

function bigBlindLog(game: GameType) {
    let user = game.players[game.bb].username;
    let bb = game.config.bb;
    return user + " is big blind (" + bb + ")";
}

function smallBlindLog(game: GameType) {
    let user = game.players[game.sb].username;
    let sb = game.config.sb;
    return user + " is small blind (" + sb + ")";
}

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
        // this effect triggers at start of every new hand
        if (game && game.running && game.stage === 2 && socket) {
            sendLog(socket, dealerLog(game));
            sendLog(socket, bigBlindLog(game));
            sendLog(socket, smallBlindLog(game));
        }
    }, [game?.stage]);

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
        <div className="mx-24  flex h-screen justify-center">
            <div className="relative mt-36 flex h-1/2 w-5/6 max-w-[800px] flex-col items-center justify-center rounded-full bg-green-600">
                <div className="flex w-full flex-col items-center justify-center">
                    <Pot game={appState.game} />
                    <div className="mt-8 mb-20 flex w-full items-center justify-center">
                        <CommunityCards />
                    </div>
                </div>
                {players.map((player, index) => (
                    <Seat
                        key={index}
                        player={player}
                        id={index + 1}
                        reveal={player ? revealedPlayers.includes(player) : false}
                    />
                ))}
            </div>
        </div>
    );
}
