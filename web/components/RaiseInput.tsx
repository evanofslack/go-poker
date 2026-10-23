import { useState } from "react";
import { Betting, clampRaise, potRaiseTo } from "../lib/betting";
import Button from "./Button";

type raiseProps = {
    b: Betting;
    onRaise: (to: number) => void;
    onClose: () => void;
    step: number; // the arrow keys change the amount by this
};

export default function RaiseInput({ b, onRaise, onClose, step }: raiseProps) {
    const [text, setText] = useState(String(b.minTo));

    const typed = parseInt(text);
    const amount = clampRaise(b, isNaN(typed) ? b.minTo : typed);
    const clamped = text !== "" && typed !== amount;

    const presets: [string, number][] = [
        ["min", b.minTo],
        ["½ pot", potRaiseTo(b, 0.5)],
        ["pot", potRaiseTo(b, 1)],
        ["all in", b.maxTo],
    ];

    return (
        <form
            className="relative flex w-full items-end gap-2 sm:w-auto"
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
                    setText(String(clampRaise(b, amount + (e.key === "ArrowUp" ? step : -step))));
                }
            }}
        >
            {/* the presets and slider float above the footer, so it keeps its height */}
            <div className="absolute right-0 bottom-full left-0 z-40 mb-2 flex flex-col items-center rounded-md border-2 border-line bg-surface px-2 sm:left-auto">
                <div className="flex w-full flex-row">
                    {presets.map(([label, to]) => (
                        <button
                            key={label}
                            type="button"
                            className="mx-0.5 my-1.5 flex-1 rounded-md border-2 border-line p-1.5 text-sm font-light whitespace-nowrap text-fg hover:bg-line sm:p-2 sm:text-base"
                            onClick={() => setText(String(clampRaise(b, to)))}
                        >
                            {label}
                        </button>
                    ))}
                </div>
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
            <input
                autoFocus
                className="w-24 rounded-md border-2 border-line bg-surface p-2 text-lg text-fg focus:outline-hidden sm:w-28 sm:text-xl"
                id="raise"
                inputMode="numeric"
                value={text}
                onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
            />
            <Button variant="primary" type="submit" className="flex-1 sm:flex-none sm:text-xl">
                {b.opening ? `bet ${amount}` : `raise to ${amount}`}
            </Button>
            <Button type="button" className="sm:text-xl" onClick={onClose}>
                close
            </Button>
        </form>
    );
}
