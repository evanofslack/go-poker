import { useState } from "react";
import classNames from "classnames";
import { Betting, clampRaise, potRaiseTo } from "../lib/betting";
import { tall, useMedia } from "../hooks/useMedia";
import { ActionIcon, Button, Slider, TextInput } from "@mantine/core";
import { bar } from "../lib/theme";

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
    // steppers, at the ends, stand in for the slider on a tall screen
    const stepper = (label: string, d: number, sign: string) =>
        onTall && (
            <ActionIcon variant="default" size="input-sm" aria-label={label} onClick={() => by(d)}>
                {sign}
            </ActionIcon>
        );
    const presetRow = (
        <div className={classNames("flex w-full flex-row gap-1", !onTall && "my-1.5")}>
            {stepper("less", -step, "−")}
            {presets.map(([label, to]) => (
                <Button
                    key={label}
                    size={onTall ? "sm" : "xs"}
                    className="flex-1 px-1 font-normal"
                    onClick={() => setText(String(clampRaise(b, to)))}
                >
                    {label}
                </Button>
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
                if ((e.target as HTMLElement).getAttribute("role") === "slider") return;
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
                <div className="absolute right-0 bottom-full left-0 z-40 mb-2 flex flex-col items-center rounded-md border border-line bg-surface px-2 sm:left-auto">
                    {presetRow}
                    <Slider
                        className="w-full sm:w-64"
                        color="gray"
                        label={null}
                        value={amount}
                        onChange={(v) => setText(String(v))}
                        min={b.minTo}
                        max={b.maxTo}
                    />
                    <div className="mt-1 flex w-full flex-row justify-between gap-2 pb-1 text-xs text-subtle sm:w-64">
                        <span>min {b.minTo}</span>
                        {clamped && <span>{`${b.opening ? "bet" : "raise"} is ${amount}`}</span>}
                        <span>max {b.maxTo}</span>
                    </div>
                </div>
            )}
            <div className="flex items-end gap-2">
                <TextInput
                    // focusing it on a phone would open the keyboard over the table
                    autoFocus={!onTall}
                    size="md"
                    className="w-24 sm:w-28"
                    classNames={{ input: "text-lg sm:h-12 sm:text-xl tall:h-12" }}
                    id="raise"
                    inputMode="numeric"
                    value={text}
                    onChange={(e) => setText(e.target.value.replace(/\D/g, ""))}
                />
                <Button variant="filled" type="submit" size="md" className={classNames(bar, "flex-1 sm:flex-none")}>
                    {b.opening ? `bet ${amount}` : `raise to ${amount}`}
                </Button>
                <Button
                    type="button"
                    size="md"
                    className={bar}
                    aria-label={onTall ? "close" : undefined}
                    onClick={onClose}
                >
                    {onTall ? "✕" : "close"}
                </Button>
            </div>
        </form>
    );
}
