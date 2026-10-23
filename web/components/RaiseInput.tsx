import { useState } from "react";
import classNames from "classnames";
import { Betting, clampRaise, potRaiseTo } from "../lib/betting";
import { tall, useMedia } from "../hooks/useMedia";
import Button from "./Button";

type raiseProps = {
    b: Betting;
    onRaise: (to: number) => void;
    onClose: () => void;
    step: number; // the arrow keys change the amount by this
};

export default function RaiseInput({ b, onRaise, onClose, step }: raiseProps) {
    const [text, setText] = useState(String(b.minTo));
    const onTall = useMedia(tall);

    const typed = parseInt(text);
    const amount = clampRaise(b, isNaN(typed) ? b.minTo : typed);
    const clamped = text !== "" && typed !== amount;

    const presets: [string, number][] = [
        ["min", b.minTo],
        ["½ pot", potRaiseTo(b, 0.5)],
        ["pot", potRaiseTo(b, 1)],
        ["all in", b.maxTo],
    ];

    const by = (d: number) => setText(String(clampRaise(b, amount + d)));
    const chip =
        "mx-0.5 rounded-md border-2 border-line p-1.5 text-sm font-light whitespace-nowrap text-fg hover:bg-line sm:p-2 sm:text-base";
    // steppers, at the ends, stand in for the slider on a tall screen
    const stepper = (label: string, d: number, sign: string) =>
        onTall && (
            <button
                type="button"
                className={classNames(chip, "px-3")}
                aria-label={label}
                onClick={() => by(d)}
            >
                {sign}
            </button>
        );
    const presetRow = (
        <div className="flex w-full flex-row">
            {stepper("less", -step, "−")}
            {presets.map(([label, to]) => (
                <button
                    key={label}
                    type="button"
                    className={classNames(chip, "flex-1", !onTall && "my-1.5")}
                    onClick={() => setText(String(clampRaise(b, to)))}
                >
                    {label}
                </button>
            ))}
            {stepper("more", step, "+")}
        </div>
    );

    return (
        <form
            className={classNames("relative flex w-full gap-2 sm:w-auto", onTall && "flex-col")}
            onSubmit={(e) => {
                e.preventDefault();
                onRaise(amount);
            }}
            onKeyDown={(e) => {
                if (e.key === "Escape") onClose();
                // the slider handles its own arrows
                if ((e.target as HTMLInputElement).type === "range") return;
                if (e.key === "ArrowUp" || e.key === "ArrowDown") {
                    e.preventDefault();
                    by(e.key === "ArrowUp" ? step : -step);
                }
            }}
        >
            {/* on a tall screen the footer has room for the presets, and steppers stand in for
                the slider. Elsewhere the presets and slider float above the footer, so it keeps
                its height. */}
            {onTall ? (
                presetRow
            ) : (
                <div className="absolute right-0 bottom-full left-0 z-40 mb-2 flex flex-col items-center rounded-md border-2 border-line bg-surface px-2 sm:left-auto">
                    {presetRow}
                    <input
                        type="range"
                        className="w-full cursor-pointer accent-zinc-400 sm:w-64"
                        value={amount}
                        onChange={(e) => setText(e.target.value)}
                        min={b.minTo}
                        max={b.maxTo}
                    />
                    <div className="flex w-full flex-row justify-between gap-2 pb-1 text-xs text-subtle sm:w-64">
                        <span>min {b.minTo}</span>
                        {clamped && <span>{`${b.opening ? "bet" : "raise"} is ${amount}`}</span>}
                        <span>max {b.maxTo}</span>
                    </div>
                </div>
            )}
            <div className="flex items-end gap-2">
                <input
                    // focusing it on a phone would open the keyboard over the table
                    autoFocus={!onTall}
                    className="w-24 rounded-md border-2 border-line bg-surface p-2 text-lg text-fg focus:outline-hidden sm:w-28 sm:text-xl"
                    id="raise"
                    inputMode="numeric"
                    value={text}
                    onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                />
                <Button
                    variant="primary"
                    type="submit"
                    className="flex-1 whitespace-nowrap sm:flex-none sm:text-xl"
                >
                    {b.opening ? `bet ${amount}` : `raise to ${amount}`}
                </Button>
                <Button
                    type="button"
                    className="sm:text-xl"
                    aria-label={onTall ? "close" : undefined}
                    onClick={onClose}
                >
                    {onTall ? "✕" : "close"}
                </Button>
            </div>
        </form>
    );
}
