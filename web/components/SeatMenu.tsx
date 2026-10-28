import { useState } from "react";
import { leaveSeat, rebuy, sitOut, straddlePreset } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { PlayerView, View } from "../interfaces/index";
import BuyIn from "./BuyIn";
import Toggle from "./Toggle";
import { ActionIcon, Menu } from "@mantine/core";
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
    const close = () => {
        setOpen(false);
        setRebuying(false);
    };

    const betweenHands = view.street === "";

    if (!socket) return null;

    const sittingOut = player.sittingOut && player.stack > 0;
    const room = view.settings.maxBuy - player.stack;
    // topping up can be any amount, but a busted player buys back in
    const minRebuy = player.stack === 0 ? view.settings.minBuy : 1;

    let sitLabel = "sit out";
    if (sittingOut) sitLabel = "sit in";
    else if (!betweenHands) sitLabel = view.sitOutNext ? "play next hand" : "sit out next hand";

    return (
        <Menu
            opened={open}
            onChange={(o) => (o ? setOpen(true) : close())}
            position="top-end"
            // wider while it holds the rebuy form
            width={rebuying ? 256 : 192}
            transitionProps={{ duration: 0 }}
        >
            <Menu.Target>
                {/* the plate changes color as the hand goes, so the icon takes the plate's text color */}
                <ActionIcon
                    variant="transparent"
                    size="sm"
                    className="absolute top-1/2 right-1 z-20 -translate-y-1/2 text-inherit opacity-60 hover:opacity-100"
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
                            <Menu.Item onClick={() => sitOut(socket)}>{sitLabel}</Menu.Item>
                        )}
                        {betweenHands && room >= minRebuy && !view.ended && (
                            <Menu.Item closeMenuOnClick={false} onClick={() => setRebuying(true)}>
                                rebuy
                            </Menu.Item>
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
                        <Menu.Item color="red" onClick={() => leaveSeat(socket)}>
                            leave seat
                        </Menu.Item>
                    </>
                )}
            </Menu.Dropdown>
        </Menu>
    );
}
