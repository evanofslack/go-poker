import { useState, useContext, useEffect, useCallback } from "react";
import { AppContext } from "../providers/AppStore";
import { playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";

type raiseProps = {
    showRaise: boolean;
    setShowRaise: React.Dispatch<React.SetStateAction<boolean>>;
};

export default function RaiseInput({ showRaise, setShowRaise }: raiseProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    if (!appState.game) {
        return null;
    }

    const playerBets = appState.game.players.map((player) => player.bet);
    const maxBet = Math.max(...playerBets);
    const minRaise = maxBet + appState.game.minRaise;
    const [inputValue, setInputValue] = useState(minRaise);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const bet = parseInt(e.target.value);
        setInputValue(bet);
    }, []);

    const handleRaise = (amount: number) => {
        if (socket) {
            playerRaise(socket, amount);
        }
    };

    return (
        <div className="flex flex-row p-6">
            <input
                className="m-4 rounded-sm bg-gray-600 p-1 text-white"
                id="input"
                type="text"
                value={inputValue ? inputValue : ""}
                onChange={handleChange}
            />
            <InputButton action={() => handleRaise(inputValue)} title={"bet"} disabled={false} />
            <InputButton action={() => setShowRaise(!showRaise)} title={"close"} disabled={false} />
        </div>
    );
}
