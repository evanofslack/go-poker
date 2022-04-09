import { useState, useContext, useEffect, useCallback } from "react";
import { AppContext } from "../providers/AppStore";
import { playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";
import { Slider } from "@mantine/core";
import classNames from "classnames";

type raiseProps = {
    showRaise: boolean;
    setShowRaise: React.Dispatch<React.SetStateAction<boolean>>;
};

function button() {
    return classNames(
        "mx-0.5 my-2 rounded-sm  border border-2 border-zinc-600  p-2 text-neutral-200 hover:bg-zinc-600 font-normal"
    );
}

export default function RaiseInput({ showRaise, setShowRaise }: raiseProps) {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);

    if (!appState.game) {
        return null;
    }

    const currentBet = appState.game.players[appState.game.action].bet;
    const playerBets = appState.game.players.map((player) => player.bet);
    const bigBlind = appState.game.config.bb;
    const maxBet = Math.max(...playerBets);
    const minRaise = maxBet + appState.game.minRaise;
    const stack = appState.game.players[appState.game.action].stack;
    const half =
        appState.game.pots.length != 0
            ? Math.ceil((appState.game.pots[0].amount + maxBet) / 2)
            : minRaise;
    const threeFourth =
        appState.game.pots.length != 0
            ? Math.ceil(((appState.game.pots[0].amount + maxBet) * 3) / 4)
            : Math.ceil(bigBlind * 2.5);

    const fullPot =
        appState.game.pots.length != 0 ? appState.game.pots[0].amount + maxBet : bigBlind * 3;
    const allIn = stack;
    const [inputValue, setInputValue] = useState(minRaise);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const bet = parseInt(e.target.value);
        setInputValue(bet);
    }, []);

    const handleRaise = (amount: number) => {
        if (socket) {
            playerRaise(socket, amount);
        }
        setShowRaise(!showRaise);
    };

    return (
        <div className="flex flex-row items-end p-6">
            <input
                autoFocus
                className="mx-1 w-24 rounded-md border border-2 border-zinc-600 bg-zinc-800 p-2 text-2xl font-medium text-neutral-200 focus:outline-none"
                id="input"
                type="text"
                value={inputValue ? inputValue : ""}
                onChange={handleChange}
            />
            <div className="mx-1 flex flex-col items-center justify-center rounded-md border border-2 border-zinc-600 px-2">
                <div className="flex flex-row items-center justify-between">
                    <button className={button()} onClick={() => setInputValue(minRaise)}>
                        min{" "}
                    </button>
                    <button
                        className={button()}
                        onClick={() => setInputValue(half >= minRaise ? half : minRaise)}
                    >
                        1/2 pot
                    </button>
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(threeFourth >= minRaise ? threeFourth : minRaise)
                        }
                    >
                        3/4 pot
                    </button>
                    <button
                        className={button()}
                        onClick={() => setInputValue(fullPot >= minRaise ? fullPot : minRaise)}
                    >
                        pot
                    </button>
                    <button className={button()} onClick={() => setInputValue(allIn)}>
                        all in
                    </button>
                </div>
                <div className="w-64 pb-2">
                    <Slider
                        value={inputValue}
                        onChange={setInputValue}
                        min={minRaise}
                        max={stack}
                        step={1}
                        color="gray"
                        showLabelOnHover={false}
                        size="md"
                        radius="xs"
                    />
                </div>
            </div>
            <InputButton
                action={() => handleRaise(inputValue - currentBet)}
                title={"bet"}
                disabled={inputValue < minRaise || inputValue > stack}
            />
            <InputButton action={() => setShowRaise(!showRaise)} title={"close"} disabled={false} />
        </div>
    );
}
