import Seat from "./Seat";
import Felt from "./Felt";
import { Player } from "../interfaces";
import { AppContext } from "../providers/AppStore";
import React, { useContext, useEffect } from "react";

type tableProps = {
    players: (Player | null)[];
    setPlayers: React.Dispatch<React.SetStateAction<(Player | null)[]>>;
};

export default function Table({ players, setPlayers }: tableProps) {
    const { appState } = useContext(AppContext);
    const game = appState.game;

    // map game players to their visual seats
    useEffect(() => {
        // start empty so seats of players who left are cleared
        let updatedPlayers: (Player | null)[] = players.map(() => null);
        if (game?.players == null) {
            return;
        }
        for (let i = 0; i < game.players.length; i++) {
            updatedPlayers[game.players[i].seatID - 1] = game.players[i];
        }
        setPlayers(updatedPlayers);
    }, [game?.players]);

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
