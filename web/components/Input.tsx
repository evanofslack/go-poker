import { useState, useContext, useEffect, useRef } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { bettingInfo } from "../lib/betting";
import { Pre, owed, preOptions, preStreet, resolvePre } from "../lib/preaction";
import Button from "./Button";
import RaiseInput from "./RaiseInput";

const btn = "flex-1 px-2 sm:flex-none sm:px-4 sm:text-xl";
const pill =
    "flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-md border-2 px-2 py-2 whitespace-nowrap sm:flex-none sm:px-4 sm:text-lg";

export default function Input() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);
    const [pre, setPre] = useState<Pre | null>(null);
    const sent = useRef(false);

    const view = appState.view;
    const open = appState.connection === "open";
    const me = view?.players.find((p) => p.seat === view.you);
    const turn = !!view?.legal.turn;

    // keeps a pick up to date, and plays it when your turn comes
    useEffect(() => {
        if (!turn) sent.current = false;
        if (!pre) return;
        if (!socket || !view || !me || !open || !view.betting || !me.in || pre.street !== preStreet(view)) {
            setPre(null);
            return;
        }
        const owe = turn ? view.legal.call : owed(view, me);
        const act = resolvePre(pre, owe);
        if (!act) {
            setPre(null);
        } else if (!turn) {
            // check/fold facing a bet is just a fold now
            if (pre.kind === "checkFold" && act === "fold") setPre({ ...pre, kind: "fold" });
        } else if (!sent.current) {
            sent.current = true;
            setPre(null);
            if (act === "fold") playerFold(socket);
            else if (act === "check") playerCheck(socket);
            else playerCall(socket);
        }
    }, [socket, view, me, open, turn, pre]);

    if (!socket || !view || !view.betting) return null;

    if (!open) {
        return <p className="py-2 text-subtle sm:text-xl">reconnecting…</p>;
    }

    const b = bettingInfo(view);
    if (!turn) {
        const waiting = `waiting for ${b.player.name}`;
        if (!me?.in || me.stack === 0) {
            return (
                <p aria-live="polite" className="py-2 text-subtle sm:text-xl">
                    {waiting}
                </p>
            );
        }
        const owe = owed(view, me);
        const street = preStreet(view);
        // the waiting line floats above the picks on a phone, so the footer keeps its height
        return (
            <div role="group" aria-label="before your turn" className="relative flex w-full gap-2 sm:w-auto">
                <p
                    aria-live="polite"
                    className="absolute right-0 bottom-full mb-1 text-sm whitespace-nowrap text-subtle sm:static sm:m-0 sm:self-center sm:text-xl"
                >
                    {waiting}
                </p>
                {preOptions(owe, me.stack).map(([kind, label]) => {
                    const on = pre?.kind === kind;
                    return (
                        <label
                            key={kind}
                            className={classNames(
                                pill,
                                on ? "border-accent text-fg" : "border-line text-muted hover:bg-raised"
                            )}
                        >
                            <input
                                type="checkbox"
                                className="size-4 accent-accent"
                                checked={on}
                                onChange={() => setPre(on ? null : { kind, amount: owe, street })}
                            />
                            {label}
                        </label>
                    );
                })}
            </div>
        );
    }

    const raiseTo = (to: number) => {
        playerRaise(socket, to);
        setShowRaise(false);
    };

    if (showRaise) {
        return <RaiseInput b={b} onRaise={raiseTo} onClose={() => setShowRaise(false)} />;
    }

    let callLabel = "check";
    if (b.toCall > 0) {
        callLabel = b.toCall === b.player.stack ? `all in ${b.toCall}` : `call ${b.toCall}`;
    }

    // the buttons ignore clicks for a moment, so a tap meant for a pick doesn't land on them
    return (
        <div role="group" aria-label="your turn" className="flex w-full animate-arm gap-2 sm:w-auto">
            <Button variant="danger" className={btn} onClick={() => playerFold(socket)}>
                fold
            </Button>
            <Button
                variant="primary"
                className={btn}
                onClick={() => (b.toCall > 0 ? playerCall(socket) : playerCheck(socket))}
            >
                {callLabel}
            </Button>
            {b.canRaise && b.fullRaise && (
                <Button variant="primary" className={btn} onClick={() => setShowRaise(true)}>
                    {b.opening ? "bet" : "raise"}
                </Button>
            )}
            {b.canRaise && !b.fullRaise && (
                <Button variant="primary" className={btn} onClick={() => raiseTo(b.maxTo)}>
                    all in {b.maxTo}
                </Button>
            )}
        </div>
    );
}
