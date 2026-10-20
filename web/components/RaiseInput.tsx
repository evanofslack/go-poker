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
            className="flex flex-row items-end p-6"
            onSubmit={(e) => {
                e.preventDefault();
                onRaise(amount);
            }}
            onKeyDown={(e) => e.key === "Escape" && onClose()}
        >
            <div className="mx-1 flex flex-col">
                <span className="mb-1 text-xs text-subtle">
                    {clamped ? `${b.opening ? "bet" : "raise"} is ${amount}` : " "}
                </span>
                <input
                    autoFocus
                    className="w-28 rounded-md border-2 border-zinc-600 bg-surface p-2 text-2xl text-fg focus:outline-hidden"
                    id="raise"
                    inputMode="numeric"
                    value={text}
                    onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                />
            </div>
            <div className="mx-1 flex flex-col items-center rounded-md border-2 border-zinc-600 px-2">
                <div className="flex flex-row">
                    {presets.map(([label, to]) => (
                        <button
                            key={label}
                            type="button"
                            className="mx-0.5 my-1.5 rounded-md border-2 border-zinc-600 p-2 font-light text-fg hover:bg-zinc-600"
                            onClick={() => setText(String(clampRaise(b, to)))}
                        >
                            {label}
                        </button>
                    ))}
                </div>
                <input
                    type="range"
                    className="w-64 cursor-pointer accent-zinc-400"
                    value={amount}
                    onChange={(e) => setText(e.target.value)}
                    min={b.minTo}
                    max={b.maxTo}
                />
                <div className="flex w-64 flex-row justify-between pb-1 text-xs text-subtle">
                    <span>min {b.minTo}</span>
                    <span>max {b.maxTo}</span>
                </div>
            </div>
            <Button variant="primary" type="submit" className="mx-1 text-xl">
                {b.opening ? `bet ${amount}` : `raise to ${amount}`}
            </Button>
            <Button type="button" className="mx-1 text-xl" onClick={onClose}>
                close
            </Button>
        </form>
    );
}
