import classNames from "classnames";
import Hint from "./Hint";

type swatchesProps<T> = {
    legend: string;
    hint?: string;
    name: string;
    look: "felt" | "back"; // the data attribute each swatch sets on itself
    options: readonly T[];
    value: T;
    onChange: (value: T) => void;
};

const looks = {
    felt: "rounded-full border-4 border-rail bg-felt text-felt-ink",
    back: "card-back rounded-md text-back-ink",
};

// Swatches picks one of a few colors, each shown as it looks on the table,
// with a legend like Segmented's
export default function Swatches<T extends string>({ legend, hint, name, look, options, value, onChange }: swatchesProps<T>) {
    return (
        <fieldset>
            <legend className="mb-1 flex items-center gap-1 text-xs text-muted">
                {legend}
                {hint && <Hint about={legend} text={hint} />}
            </legend>
            <div className="flex flex-wrap gap-1.5">
                {options.map((o) => (
                    <label
                        key={o}
                        {...{ [`data-${look}`]: o }}
                        className={classNames(
                            looks[look],
                            "flex h-8 w-16 cursor-pointer items-center justify-center text-xs font-medium",
                            "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent",
                            o === value && "ring-2 ring-accent ring-offset-2 ring-offset-surface"
                        )}
                    >
                        <input type="radio" className="sr-only" name={name} value={o} checked={o === value} onChange={() => onChange(o)} />
                        {o}
                    </label>
                ))}
            </div>
        </fieldset>
    );
}
