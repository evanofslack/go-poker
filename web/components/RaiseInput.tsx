import { useState } from "react";
import { Betting, clampRaise, potRaiseTo } from "../lib/betting";
import Button from "./Button";

type raiseProps = {
    b: Betting;
    onRaise: (to: number) => void;
    onClose: () => void;
};

export default function RaiseInput({ b, onRaise, onClose }: raiseProps) {
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
            className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-end"
            onSubmit={(e) => {
                e.preventDefault();
                onRaise(amount);
            }}
            onKeyDown={(e) => e.key === "Escape" && onClose()}
        >
            <div className="flex flex-col items-center rounded-md border-2 border-line px-2 sm:order-2">
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
                <div className="flex w-full flex-row justify-between pb-1 text-xs text-subtle sm:w-64">
                    <span>min {b.minTo}</span>
                    <span>max {b.maxTo}</span>
                </div>
            </div>
            <div className="flex items-end gap-2 sm:contents">
                <div className="flex flex-col sm:order-1">
                    <span className="mb-1 text-xs text-subtle">
                        {clamped ? `${b.opening ? "bet" : "raise"} is ${amount}` : "\u00a0"}
                    </span>
                    <input
                        autoFocus
                        className="w-24 rounded-md border-2 border-line bg-surface p-2 text-xl text-fg focus:outline-hidden sm:w-28 sm:text-2xl"
                        id="raise"
                        inputMode="numeric"
                        value={text}
                        onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                    />
                </div>
                <Button
                    variant="primary"
                    type="submit"
                    className="flex-1 sm:order-3 sm:flex-none sm:text-xl"
                >
                    {b.opening ? `bet ${amount}` : `raise to ${amount}`}
                </Button>
                <Button type="button" className="sm:order-4 sm:text-xl" onClick={onClose}>
                    close
                </Button>
            </div>
        </form>
    );
}
