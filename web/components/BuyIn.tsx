import { useState } from "react";
import Button from "./Button";

type buyInProps = {
    max: number;
    initial: number;
    label: string;
    onSubmit: (amount: number) => void;
    onCancel: () => void;
};

// BuyIn asks for an amount of chips, for taking a seat or rebuying
export default function BuyIn({ max, initial, label, onSubmit, onCancel }: buyInProps) {
    const [text, setText] = useState(String(Math.min(initial, max)));
    const amount = parseInt(text);
    const valid = amount >= 1 && amount <= max;

    return (
        <form
            className="flex h-full flex-col justify-center px-3"
            onSubmit={(e) => {
                e.preventDefault();
                if (valid) onSubmit(amount);
            }}
            onKeyDown={(e) => e.key === "Escape" && onCancel()}
        >
            <div className="flex flex-row items-center gap-1">
                <input
                    autoFocus
                    className="w-20 rounded-md bg-zinc-600 p-1 text-fg focus:outline-hidden"
                    id="buyIn"
                    inputMode="numeric"
                    value={text}
                    onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                />
                <Button
                    type="submit"
                    variant="primary"
                    className="px-2 py-1 text-sm whitespace-nowrap"
                    disabled={!valid}
                >
                    {label}
                </Button>
                <Button
                    type="button"
                    variant="ghost"
                    className="px-1 py-1 text-sm"
                    onClick={onCancel}
                >
                    cancel
                </Button>
            </div>
            <span className={"mt-1 text-xs " + (valid ? "text-subtle" : "text-rose-400")}>
                max {max}
            </span>
        </form>
    );
}
