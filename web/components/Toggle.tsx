import Hint from "./Hint";

type toggleProps = {
    id: string;
    label: string;
    hint?: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
};

// Toggle is an on/off setting: the label on the left, a switch on the right.
// The switch is a styled checkbox, so it checks and reads like one.
export default function Toggle({ id, label, hint, checked, onChange }: toggleProps) {
    return (
        <div className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-sm">
            <label htmlFor={id} className="cursor-pointer">
                {label}
            </label>
            {hint && <Hint about={label} text={hint} />}
            <input
                id={id}
                type="checkbox"
                role="switch"
                className="relative ml-auto h-5 w-9 shrink-0 cursor-pointer appearance-none rounded-full bg-line transition-colors before:absolute before:top-0.5 before:left-0.5 before:size-4 before:rounded-full before:bg-fg before:transition-transform checked:bg-accent checked:before:translate-x-4 checked:before:bg-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                checked={checked}
                onChange={(e) => onChange(e.target.checked)}
            />
        </div>
    );
}
