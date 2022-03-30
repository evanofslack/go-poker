import { useContext, useState, useEffect } from "react";
import Chat from "./Chat";
import Seat from "./Seat";
import CommunityCards from "./CommunityCards";
import Input from "./Input";
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
            updatedPlayers[appState.game.players[i].position - 1] = appState.game.players[i];
        }
        setPlayers(updatedPlayers);
    }, [appState.game?.players]);

    return (
        <div className="flex h-screen flex-row items-start justify-center">
            <div className="mx-24 mt-24 h-3/5 w-5/6 items-center justify-center rounded-3xl bg-green-600">
                <div className="flex w-full items-center justify-center">
                    <CommunityCards />
                </div>
                <div className="flex flex-row flex-wrap justify-center">
                    {players.map((player, index) => (
                        <Seat key={index} player={player} id={index + 1} />
                    ))}
                </div>
            </div>
            <div className="absolute left-0 bottom-0">
                <Chat />
            </div>
            <div className="absolute bottom-36 right-10">
                <Input />
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
