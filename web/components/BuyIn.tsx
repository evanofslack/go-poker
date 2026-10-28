import { useState } from "react";
import { Button, TextInput } from "@mantine/core";

type buyInProps = {
    min: number;
    max: number;
    initial: number;
    label: string;
    onSubmit: (amount: number) => void;
    onCancel: () => void;
};

// BuyIn asks for an amount of chips, for taking a seat or rebuying
export default function BuyIn({ min, max, initial, label, onSubmit, onCancel }: buyInProps) {
    const [text, setText] = useState(String(Math.max(min, Math.min(initial, max))));
    const amount = parseInt(text);
    const valid = amount >= Math.max(min, 1) && amount <= max;

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
                <TextInput
                    autoFocus
                    size="xs"
                    className="w-20 min-w-16 flex-1"
                    id="buyIn"
                    inputMode="numeric"
                    value={text}
                    onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                />
                <Button type="submit" variant="filled" size="compact-sm" className="shrink-0" disabled={!valid}>
                    {label}
                </Button>
                <Button type="button" variant="subtle" size="compact-sm" className="shrink-0" onClick={onCancel}>
                    cancel
                </Button>
            </div>
            <span className={"mt-1 text-xs " + (valid ? "text-subtle" : "text-error")}>
                {min > 1 ? `${min} to ${max}` : `max ${max}`}
            </span>
        </form>
    );
}
