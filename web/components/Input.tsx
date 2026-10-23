import { useState, useContext, useEffect, useRef } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { playerCall, playerCheck, playerFold, playerRaise } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { bettingInfo } from "../lib/betting";
import { Pre, owed, preOptions, preStreet, resolvePre } from "../lib/preaction";
import Button from "./Button";
import RaiseInput from "./RaiseInput";

const btn = "flex-1 px-2 sm:flex-none sm:px-4 sm:text-xl tall:py-3";
// key hints, drawn after the label so they aren't part of its text
const hint =
    "pointer-fine:after:ml-2 pointer-fine:after:text-xs pointer-fine:after:font-normal pointer-fine:after:opacity-60";
const hints: Record<string, string> = {
    F: "pointer-fine:after:content-['F']",
    C: "pointer-fine:after:content-['C']",
    R: "pointer-fine:after:content-['R']",
};

// typing somewhere, like the chat, isn't a shortcut
function typing(t: EventTarget | null) {
    if (!(t instanceof HTMLElement)) return false;
    if (t instanceof HTMLInputElement) return !["checkbox", "radio", "button", "submit", "range"].includes(t.type);
    return t instanceof HTMLTextAreaElement || t instanceof HTMLSelectElement || t.isContentEditable;
}

const pill =
    "flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-md border-2 px-2 py-2 whitespace-nowrap sm:flex-none sm:px-4 sm:text-lg";

export default function Input() {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [showRaise, setShowRaise] = useState(false);
    const [pre, setPre] = useState<Pre | null>(null);
    const sent = useRef(false);
    const turnAt = useRef(0);
    const onKey = useRef<(key: string) => boolean>(() => false);

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

    useEffect(() => {
        if (turn) turnAt.current = Date.now();
    }, [turn]);

    // F and C act on your turn, or pick before it; R and B open the bet
    onKey.current = (key) => {
        if (!appState.shortcuts || !socket || !view?.betting || !open || !me?.in || me.stack === 0) return false;
        if (!turn) {
            const owe = owed(view, me);
            const kind = preOptions(owe, me.stack)[key === "f" ? 0 : key === "c" ? 1 : -1]?.[0];
            if (!kind) return false;
            setPre(pre?.kind === kind ? null : { kind, amount: owe, street: preStreet(view) });
            return true;
        }
        // the raise panel has its own keys, and the buttons ignore input as they appear
        if (showRaise || Date.now() - turnAt.current < 300) return false;
        const b = bettingInfo(view);
        if (key === "f" && b.toCall > 0) playerFold(socket);
        else if (key === "c") b.toCall > 0 ? playerCall(socket) : playerCheck(socket);
        else if ((key === "r" || key === "b") && b.canRaise && b.fullRaise) setShowRaise(true);
        else return false;
        return true;
    };

    useEffect(() => {
        const key = (e: KeyboardEvent) => {
            if (e.repeat || e.metaKey || e.ctrlKey || e.altKey || typing(e.target)) return;
            if (document.querySelector('[aria-modal="true"]')) return;
            if (onKey.current(e.key.toLowerCase())) e.preventDefault();
        };
        document.addEventListener("keydown", key);
        return () => document.removeEventListener("keydown", key);
    }, []);

    const keyed = (k: string) => (appState.shortcuts ? classNames(hint, hints[k]) : "");

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
                {preOptions(owe, me.stack).map(([kind, label], i) => {
                    const on = pre?.kind === kind;
                    const k = i === 0 ? "F" : "C";
                    return (
                        <label
                            key={kind}
                            className={classNames(
                                pill,
                                keyed(k),
                                on ? "border-accent text-fg" : "border-line text-muted hover:bg-raised"
                            )}
                            aria-keyshortcuts={k}
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
        return (
            <RaiseInput b={b} step={view.settings.bb} onRaise={raiseTo} onClose={() => setShowRaise(false)} />
        );
    }

    let callLabel = "check";
    if (b.toCall > 0) {
        callLabel = b.toCall === b.player.stack ? `all in ${b.toCall}` : `call ${b.toCall}`;
    }

    // the buttons ignore clicks for a moment, so a tap meant for a pick doesn't land on them
    return (
        <div role="group" aria-label="your turn" className="flex w-full animate-arm gap-2 sm:w-auto">
            <Button
                variant="danger"
                className={classNames(btn, b.toCall > 0 && keyed("F"))}
                aria-keyshortcuts={b.toCall > 0 ? "F" : undefined}
                onClick={() => playerFold(socket)}
            >
                fold
            </Button>
            <Button
                variant="primary"
                className={classNames(btn, keyed("C"))}
                aria-keyshortcuts="C"
                onClick={() => (b.toCall > 0 ? playerCall(socket) : playerCheck(socket))}
            >
                {callLabel}
            </Button>
            {b.canRaise && b.fullRaise && (
                <Button
                    variant="primary"
                    className={classNames(btn, keyed("R"))}
                    aria-keyshortcuts="R B"
                    onClick={() => setShowRaise(true)}
                >
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
