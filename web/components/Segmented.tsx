import { ReactNode } from "react";
import Hint from "./Hint";

type segmentedProps<T> = {
    legend: string;
    hint?: string;
    name: string;
    options: { label: ReactNode; value: T }[];
    value: T;
    onChange: (value: T) => void;
};

// Segmented picks one of a few options. They're radio buttons underneath, so
// arrow keys move between them.
export default function Segmented<T extends string | number>({
    legend,
    hint,
    name,
    options,
    value,
    onChange,
}: segmentedProps<T>) {
    return (
        <fieldset>
            <legend className="mb-1 flex w-full flex-wrap items-center gap-x-1 gap-y-0.5 text-xs text-muted">
                {legend}
                {hint && <Hint about={legend} text={hint} />}
            </legend>
            <div className="flex flex-wrap gap-1">
                {options.map((o) => (
                    <label key={o.value} className="cursor-pointer">
                        <input
                            type="radio"
                            className="peer sr-only"
                            name={name}
                            checked={o.value === value}
                            onChange={() => onChange(o.value)}
                        />
                        <span className="block rounded-md border-2 border-line px-2 py-0.5 text-sm hover:bg-raised peer-checked:border-accent peer-checked:bg-accent peer-checked:font-semibold peer-checked:hover:bg-accent peer-checked:text-on-accent peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent">
                            {o.label}
                        </span>
                    </label>
                ))}
            </div>
        </fieldset>
    );
}
