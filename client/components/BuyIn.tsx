import React, { useState, useContext, useCallback } from "react";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { FcCheckmark } from "react-icons/fc";

type buyInProps = {
    seatID: number;
    sitDown: boolean;
    setSitDown: React.Dispatch<React.SetStateAction<boolean>>;
};

export default function BuyIn({ seatID, sitDown, setSitDown }: buyInProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [buyIn, setBuyIn] = useState(
        appState.game?.config.maxBuyIn ? appState.game?.config.maxBuyIn : 2000
    );

    const handleBuyIn = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const amount = parseInt(e.target.value);
        setBuyIn(amount);
    }, []);

    const handleClick = () => {
        if (socket && appState.username) {
            takeSeat(socket, appState.username, seatID, buyIn);
            setSitDown(!sitDown);
        }
    };
    return (
        <div className="relative right-2 m-4 flex h-full w-full flex-row items-center justify-center">
            <p className="-mb-1 text-lg font-normal">{appState.username}</p>
            <div className="flex flex-row  pl-8">
                <input
                    autoFocus
                    className="mr-2 w-20 rounded-sm bg-neutral-600 p-1 text-white focus:outline-none"
                    id="buyIn"
                    type="number"
                    value={buyIn}
                    onChange={handleBuyIn}
                />
                <button onClick={handleClick}>
                    <FcCheckmark size="1.7rem" />
                </button>
            </div>
        </div>
    );
}
