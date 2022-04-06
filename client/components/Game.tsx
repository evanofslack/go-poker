import { useContext, useState, useEffect } from "react";
import Chat from "./Chat";
import Seat from "./Seat";
import CommunityCards from "./CommunityCards";
import Input from "./Input";
import { useSocket } from "../hooks/useSocket";
import { startGame, sendMessage, dealGame } from "../actions/actions";
import { AppContext } from "../providers/AppStore";
import { Game as GameType, Player } from "../interfaces";

function sleep(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

function handleWinner(game: GameType | null, socket: WebSocket | null) {
    if (!game || !socket) {
        return null;
    }
    if (game && game.stage === 1 && game.pots.length !== 0) {
        const winnerNum = game.pots[game.pots.length - 1].winningPlayerNums[0];
        const winningPlayer = game.players.filter((player) => player.position == winnerNum);
        const pot = game.pots[game.pots.length - 1].amount;
        console.log(winningPlayer);
        const message = winningPlayer[0].username + " wins " + pot;
        sendMessage(socket, "system", message);
        dealGame(socket);
    }
}

function handleStartGame(socket: WebSocket | null) {
    if (socket) {
        startGame(socket);
    }
}

export default function Game() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    const game = appState.game;

    const initialPlayers: (Player | null)[] = [null, null, null, null, null, null];
    const [players, setPlayers] = useState(initialPlayers);

    // map game players to their visual seats
    useEffect(() => {
        const updatedPlayers: (Player | null)[] = [...players];
        if (game?.players == null) {
            return;
        }
        for (let i = 0; i < game.players.length; i++) {
            updatedPlayers[game.players[i].seatID - 1] = game.players[i];
        }
        setPlayers(updatedPlayers);
    }, [game?.players]);

    useEffect(() => {
        console.log(game);
    }, [appState.game?.players]);

    useEffect(() => {
        const timer = setTimeout(() => handleWinner(game, socket), 4000);
        return () => clearTimeout(timer);
    }, [game?.pots]);

    return (
        <div className="mx-24 mt-36 flex h-screen justify-center">
            <div className="relative flex h-1/2 w-5/6 max-w-[900px] flex-col items-center justify-center rounded-full bg-green-600">
                <div className="flex w-full items-center justify-center">
                    <CommunityCards />
                </div>
                <div className="flex w-full items-center justify-center">
                    {game?.pots?.map((p, index) => (
                        <div
                            key={index}
                            className="flex h-8 w-12 items-center justify-center rounded-3xl bg-amber-300 text-xl font-semibold text-black"
                        >
                            {p.amount}
                        </div>
                    ))}
                </div>
                {players.map((player, index) => (
                    <Seat key={index} player={player} id={index + 1} />
                ))}
            </div>
            <div className="absolute left-0 bottom-0">
                <Chat />
            </div>
            <div className="absolute bottom-0 right-10">
                <Input />
            </div>
            {!game?.running && (
                <button
                    className="absolute right-0 bottom-0 m-10 rounded-md bg-green-800 p-2 text-white"
                    onClick={() => handleStartGame(socket)}
                >
                    Start
                </button>
            )}
        </div>
    );
}
