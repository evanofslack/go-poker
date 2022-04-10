import { useState, useContext, useEffect, useCallback } from "react";
import { AppContext } from "../providers/AppStore";
import { playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import InputButton from "./InputButton";
import { Slider } from "@mantine/core";
import classNames from "classnames";
import { Game } from "../interfaces/index";

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

    const bigBlind = appState.game.config.bb;
    const smallBlind = appState.game.config.sb;
    const currentBet = appState.game.players[appState.game.action].bet; // active player's bet
    const currentStack = appState.game.players[appState.game.action].stack; // active player's stack
    const playerBets = appState.game.players.map((player) => player.bet); // array of all players' bets
    const maxBet = Math.max(...playerBets); // largest bet out of all player's bets
    const minRaise = maxBet + appState.game.minRaise;

    const currentPot =
        appState.game.pots.length != 0 ? appState.game.pots[0].amount : bigBlind + smallBlind;

    // a pot sized bet is equal to 3 times the previous largest bet + pot before previous bet
    const potBet = 3 * maxBet + currentPot - maxBet;

    function potPortion(pot: number, fraction: number) {
        // returns rounded fraction of the pot
        return Math.ceil(pot * fraction);
    }

    function betValidator(bet: number, minRaise: number, stack: number) {
        // bet can never be smaller than min raise and can never be bigger than player stack + committed chips
        if (bet < minRaise) {
            return minRaise;
        } else if (bet > stack) {
            return stack;
        } else {
            return bet;
        }
    }

    const half = appState.game.pots.length != 0 ? potPortion(potBet, 0.5) : minRaise;
    const threeQuarter =
        appState.game.pots.length != 0 ? potPortion(potBet, 0.75) : Math.ceil(bigBlind * 2.5);

    const full = appState.game.pots.length != 0 ? potBet : bigBlind * 3;
    const allIn = currentStack + currentBet;

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
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(
                                betValidator(minRaise, minRaise, currentStack + currentBet)
                            )
                        }
                    >
                        min{" "}
                    </button>
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(betValidator(half, minRaise, currentStack + currentBet))
                        }
                    >
                        1/2 pot
                    </button>
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(
                                betValidator(threeQuarter, minRaise, currentStack + currentBet)
                            )
                        }
                    >
                        3/4 pot
                    </button>
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(betValidator(full, minRaise, currentStack + currentBet))
                        }
                    >
                        pot
                    </button>
                    <button
                        className={button()}
                        onClick={() =>
                            setInputValue(betValidator(allIn, minRaise, currentStack + currentBet))
                        }
                    >
                        all in
                    </button>
                </div>
                <div className="w-64 pb-2">
                    <Slider
                        value={inputValue}
                        onChange={setInputValue}
                        min={minRaise}
                        max={currentStack + currentBet}
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
                disabled={inputValue < minRaise || inputValue > currentStack + currentBet}
            />
            <InputButton action={() => setShowRaise(!showRaise)} title={"close"} disabled={false} />
        </div>
    );
}
