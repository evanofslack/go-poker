import { useContext, useState, useEffect } from "react";
import Chat from "./Chat";
import Seat from "./Seat";
import { useSocket } from "../hooks/useSocket";
import { startGame } from "../actions/actions";
import { AppContext } from "../providers/AppStore";
import { Player } from "../interfaces";

export default function Game() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const handleStartGame = () => {
        if (socket) {
            startGame(socket);
        }
    };

    const initialPlayers: (Player | null)[] = [null, null, null, null, null, null];
    const [players, setPlayers] = useState(initialPlayers);

    useEffect(() => {
        const updatedPlayers: (Player | null)[] = [...players];
        if (appState.game?.players == null) {
            return;
        }
        for (let i = 0; i < appState.game.players.length; i++) {
            updatedPlayers[appState.game.players[i].position] = appState.game.players[i];
        }
        console.log(updatedPlayers);
        setPlayers(updatedPlayers);
    }, [appState.game?.players]);

    return (
        <div className="flex h-screen flex-row items-center justify-center">
            <div className="m-24 flex h-1/2 w-3/5 items-center justify-center rounded-3xl bg-green-600">
                <div className="flex flex-col">
                    {players.map((player, index) => (
                        <Seat key={index} player={player} id={index} />
                    ))}
                </div>
            </div>
            <div className="absolute left-0 bottom-0">
                <Chat />
            </div>
            <button
                className="absolute right-0 bottom-0 m-10 rounded-md bg-green-800 p-2 text-white"
                onClick={handleStartGame}
            >
                Start Game
            </button>
        </div>
    );
}
