import { ReactNode } from "react";
import { SegmentedControl } from "@mantine/core";
import Hint from "./Hint";

type segmentedProps<T> = {
    legend: string;
    hint?: string;
    name: string;
    options: { label: ReactNode; value: T }[];
    value: T;
    onChange: (value: T) => void;
    disabled?: boolean;
};

// Segmented picks one of a few options, with a legend. Mantine's values are
// strings, so it maps them back to the options' own.
export default function Segmented<T extends string | number>({
    legend,
    hint,
    name,
    options,
    value,
    onChange,
    disabled,
}: segmentedProps<T>) {
    return (
        <fieldset>
            <legend className="mb-1 flex items-center gap-1 text-xs text-muted">
                {legend}
                {hint && <Hint about={legend} text={hint} />}
            </legend>
            <SegmentedControl
                name={name}
                size="xs"
                color="cyan"
                className="bg-panel"
                // the pick moves at once, as dialogs open and close
                transitionDuration={0}
                data={options.map((o) => ({ label: o.label, value: String(o.value) }))}
                value={String(value)}
                disabled={disabled}
                onChange={(v) => onChange(options.find((o) => String(o.value) === v)!.value)}
            />
        </fieldset>
    );
}
