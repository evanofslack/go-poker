import { useRef, useState } from "react";
import { leaveSeat, rebuy, sitOut, straddlePreset } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { useDismiss } from "../hooks/useDismiss";
import { PlayerView, View } from "../interfaces/index";
import BuyIn from "./BuyIn";
import Toggle from "./Toggle";
import { ActionIcon } from "@mantine/core";
import { FiMoreHorizontal } from "react-icons/fi";

type seatMenuProps = {
    player: PlayerView;
    view: View;
};

// SeatMenu holds your own seat's controls: leave, sit out or in, rebuy, and
// whether to straddle
export default function SeatMenu({ player, view }: seatMenuProps) {
    const socket = useSocket();
    const [open, setOpen] = useState(false);
    const [rebuying, setRebuying] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    const close = () => {
        setOpen(false);
        setRebuying(false);
    };
    useDismiss(ref, open, close);

    const betweenHands = view.street === "";

    if (!socket) return null;

    const sittingOut = player.sittingOut && player.stack > 0;
    const room = view.settings.maxBuy - player.stack;
    // topping up can be any amount, but a busted player buys back in
    const minRebuy = player.stack === 0 ? view.settings.minBuy : 1;

    let sitLabel = "sit out";
    if (sittingOut) sitLabel = "sit in";
    else if (!betweenHands) sitLabel = view.sitOutNext ? "play next hand" : "sit out next hand";

    const item = "px-3 py-1.5 text-left text-sm hover:bg-raised";
    return (
        <div ref={ref}>
            {/* the plate changes color as the hand goes, so the icon takes the plate's text color */}
            <ActionIcon
                variant="transparent"
                size="sm"
                className="absolute top-1/2 right-1 z-20 -translate-y-1/2 text-inherit opacity-60 hover:opacity-100"
                title="seat options"
                aria-label="seat options"
                aria-haspopup="menu"
                aria-expanded={open}
                onClick={() => (open ? close() : setOpen(true))}
            >
                <FiMoreHorizontal />
            </ActionIcon>
            {open && (
                <div
                    role="menu"
                    className="absolute right-0 bottom-full z-30 mb-1 flex w-48 flex-col rounded-md border border-line bg-panel py-1 text-fg shadow-xl"
                >
                    {rebuying ? (
                        <div className="h-20">
                            <BuyIn
                                min={minRebuy}
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
                            {player.stack > 0 && !view.ended && (
                                <button
                                    role="menuitem"
                                    className={item}
                                    onClick={() => {
                                        sitOut(socket);
                                        close();
                                    }}
                                >
                                    {sitLabel}
                                </button>
                            )}
                            {betweenHands && room >= minRebuy && !view.ended && (
                                <button
                                    role="menuitem"
                                    className={item}
                                    onClick={() => setRebuying(true)}
                                >
                                    rebuy
                                </button>
                            )}
                            {view.settings.straddle !== "off" && !view.ended && (
                                // off asks you each time
                                <div className="px-3 py-1.5">
                                    <Toggle
                                        id="alwaysStraddle"
                                        label="always straddle"
                                        checked={view.straddlePreset === "always"}
                                        onChange={(on) => straddlePreset(socket, on ? "always" : "ask")}
                                    />
                                </div>
                            )}
                            <button
                                role="menuitem"
                                className={item + " text-danger"}
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
        </div>
    );
}
