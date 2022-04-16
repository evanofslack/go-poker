import { useContext, useState, useEffect } from "react";
import ChatLog from "./ChatLog";
import Pot from "./Pot";
import Seat from "./Seat";
import Reset from "./Reset";
import Start from "./Start";
import CommunityCards from "./CommunityCards";
import Input from "./Input";
import { useSocket } from "../hooks/useSocket";
import { sendMessage, dealGame } from "../actions/actions";
import { AppContext } from "../providers/AppStore";
import { Game as GameType, Player } from "../interfaces";

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
        // todo add sendLog action
        // sendMessage(socket, "system", message);
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

export default function Game() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    const game = appState.game;

    const initialPlayers: (Player | null)[] = [null, null, null, null, null, null];
    const [players, setPlayers] = useState(initialPlayers);
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
            <div className="absolute left-0 bottom-0">
                <ChatLog />
            </div>
            <div className="absolute bottom-0 right-0">
                <Input />
            </div>
            <div className="absolute left-0 top-0">
                <Reset />
            </div>
            <div className="absolute bottom-0 right-0">
                <Start players={players} />
            </div>
        </div>
    );
}
