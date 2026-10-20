import { useEffect, useState } from "react";
import { leaveSeat, rebuy, sitOut } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { Game, Player } from "../interfaces/index";
import BuyIn from "./BuyIn";

type seatMenuProps = {
    player: Player;
    game: Game;
};

// SeatMenu holds your own seat's controls: leave, sit out or in, and rebuy
export default function SeatMenu({ player, game }: seatMenuProps) {
    const socket = useSocket();
    const [open, setOpen] = useState(false);
    const [rebuying, setRebuying] = useState(false);
    // a sit-out sent mid-hand applies when the hand ends; the server doesn't echo it back
    const [pending, setPending] = useState(false);

    const betweenHands = !game.running || game.stage === 1;
    useEffect(() => {
        if (betweenHands) setPending(false);
    }, [betweenHands]);

    if (!socket) return null;

    const sittingOut = !player.ready && player.stack > 0;
    const room = game.config.maxBuy - player.stack;
    const close = () => {
        setOpen(false);
        setRebuying(false);
    };

    let sitLabel = "sit out";
    if (sittingOut) sitLabel = "sit in";
    else if (!betweenHands) sitLabel = pending ? "play next hand" : "sit out next hand";

    const item = "px-3 py-1.5 text-left text-sm hover:bg-raised";
    return (
        <>
            <button
                className="absolute top-0 right-2 z-20 text-lg leading-none opacity-60 hover:opacity-100"
                title="seat options"
                onClick={() => (open ? close() : setOpen(true))}
            >
                ⋯
            </button>
            {open && (
                <div className="absolute top-full right-0 z-30 mt-1 flex w-48 flex-col rounded-md border border-zinc-600 bg-panel py-1 text-fg shadow-xl">
                    {rebuying ? (
                        <div className="h-20">
                            <BuyIn
                                max={room}
                                initial={room}
                                label="rebuy"
                                onSubmit={(amount) => {
                                    rebuy(socket, amount);
                                    close();
                                }}
                                onCancel={close}
                            />
                        </div>
                    ) : (
                        <>
                            {player.stack > 0 && (
                                <button className={item}
                                    onClick={() => {
                                        sitOut(socket);
                                        if (!sittingOut && !betweenHands) setPending(!pending);
                                        close();
                                    }}
                                >
                                    {sitLabel}
                                </button>
                            )}
                            {betweenHands && room > 0 && (
                                <button className={item} onClick={() => setRebuying(true)}>
                                    rebuy
                                </button>
                            )}
                            <button className={item + " text-danger"}
                                onClick={() => {
                                    leaveSeat(socket);
                                    close();
                                }}
                            >
                                leave seat
                            </button>
                        </>
                    )}
                </div>
            )}
        </>
    );
}
