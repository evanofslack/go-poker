import { useState } from "react";
import { leaveSeat, rebuy, sitOut, straddlePreset } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { PlayerView, View } from "../interfaces/index";
import BuyIn from "./BuyIn";
import Confirm from "./Confirm";
import { ActionIcon, Menu } from "@mantine/core";
import { FiCheck, FiMoreHorizontal } from "react-icons/fi";

type seatMenuProps = {
    player: PlayerView;
    view: View;
};

// a dim note at an item's right end, like when it's available or what it costs
const note = (text: string) => <span className="text-xs text-subtle">{text}</span>;

// SeatMenu holds your own seat's controls: sit out or in, top up or rebuy,
// whether to straddle, and leave
export default function SeatMenu({ player, view }: seatMenuProps) {
    const socket = useSocket();
    const [open, setOpen] = useState(false);
    const [rebuying, setRebuying] = useState(false);
    const [leaving, setLeaving] = useState(false);
    const close = () => {
        setOpen(false);
        setRebuying(false);
    };

    const betweenHands = view.street === "";

    if (!socket) return null;

    const sittingOut = player.sittingOut && player.stack > 0;
    const room = view.settings.maxBuy - player.stack;
    // topping up can be any amount, but a busted player buys back in
    const busted = player.stack === 0;
    const minRebuy = busted ? view.settings.minBuy : 1;
    const rebuyLabel = busted ? "rebuy" : "top up";
    // leaving with cards folds them, so it asks first
    const folds = !betweenHands && player.in;

    let sitLabel = "sit out";
    if (sittingOut) sitLabel = "sit in";
    else if (!betweenHands) sitLabel = view.sitOutNext ? "play next hand" : "sit out next hand";

    const always = view.straddlePreset === "always";

    return (
        <>
            <Menu
                opened={open}
                onChange={(o) => (o ? setOpen(true) : close())}
                position="top-end"
                // wider while it holds the rebuy form
                width={rebuying ? 256 : 208}
                transitionProps={{ duration: 0 }}
            >
                <Menu.Target>
                    {/* the plate changes color as the hand goes, so the icon takes the plate's text color */}
                    <ActionIcon
                        variant="transparent"
                        size="sm"
                        className="absolute top-1/2 right-1 z-20 -translate-y-1/2 text-inherit opacity-60 hover:opacity-100 aria-expanded:opacity-100 phone:size-8"
                        title="seat options"
                        aria-label="seat options"
                        // the menu only hears Escape once focus is in it
                        onKeyDown={(e) => e.key === "Escape" && close()}
                    >
                        <FiMoreHorizontal />
                    </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                    {rebuying ? (
                        <div className="h-20">
                            <BuyIn
                                min={minRebuy}
                                max={room}
                                initial={room}
                                label={rebuyLabel}
                                onSubmit={(amount) => {
                                    rebuy(socket, amount);
                                    close();
                                }}
                                // back to the menu
                                onCancel={() => setRebuying(false)}
                            />
                        </div>
                    ) : (
                        <>
                            {player.stack > 0 && !view.ended && (
                                <Menu.Item onClick={() => sitOut(socket)}>{sitLabel}</Menu.Item>
                            )}
                            {room >= minRebuy && !view.ended && (
                                // mid-hand it's there but off, so it's clear when it comes back
                                <Menu.Item
                                    disabled={!betweenHands}
                                    closeMenuOnClick={false}
                                    rightSection={note(betweenHands ? `up to ${room.toLocaleString("en")}` : "between hands")}
                                    onClick={() => setRebuying(true)}
                                >
                                    {rebuyLabel}
                                </Menu.Item>
                            )}
                            {view.settings.straddle !== "off" && !view.ended && (
                                // off asks you each time
                                // Mantine keeps its items menuitems, so the tick has words for screen readers
                                <Menu.Item
                                    closeMenuOnClick={false}
                                    rightSection={always && <FiCheck className="text-accent" aria-hidden />}
                                    onClick={() => straddlePreset(socket, always ? "ask" : "always")}
                                >
                                    always straddle
                                    {always && <span className="sr-only">, on</span>}
                                </Menu.Item>
                            )}
                            <Menu.Divider />
                            <Menu.Item
                                color="red"
                                rightSection={folds && note("folds this hand")}
                                onClick={() => (folds ? setLeaving(true) : leaveSeat(socket))}
                            >
                                leave seat
                            </Menu.Item>
                        </>
                    )}
                </Menu.Dropdown>
            </Menu>
            {leaving && (
                <Confirm
                    title="leave your seat?"
                    message="you fold this hand and cash out. you can sit down again after."
                    action="leave seat"
                    onConfirm={() => {
                        leaveSeat(socket);
                        setLeaving(false);
                    }}
                    onCancel={() => setLeaving(false)}
                />
            )}
        </>
    );
}
