type segmentedProps<T> = {
    legend: string;
    name: string;
    options: { label: string; value: T }[];
    value: T;
    onChange: (value: T) => void;
};

// Segmented picks one of a few options. They're radio buttons underneath, so
// arrow keys move between them.
export default function Segmented<T extends string | number>({
    legend,
    name,
    options,
    value,
    onChange,
}: segmentedProps<T>) {
    return (
        <fieldset>
            <legend className="mb-1 text-xs text-muted">{legend}</legend>
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
                        <span className="block rounded-md border-2 border-zinc-600 px-2 py-0.5 text-sm hover:bg-raised peer-checked:border-accent peer-checked:bg-accent peer-checked:font-semibold peer-checked:hover:bg-accent peer-checked:text-zinc-950 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent">
                            {o.label}
                        </span>
                    </label>
                ))}
            </div>
        </fieldset>
    );
}
