import { Switch } from "@mantine/core";
import Hint from "./Hint";

type toggleProps = {
    id: string;
    label: string;
    hint?: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
};

// Toggle is an on/off setting: the label on the left, a switch on the right
export default function Toggle({ id, label, hint, checked, onChange }: toggleProps) {
    return (
        <div className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-sm">
            <label htmlFor={id} className="cursor-pointer">
                {label}
            </label>
            {hint && <Hint about={label} text={hint} />}
            <Switch
                id={id}
                className="ml-auto"
                checked={checked}
                onChange={(e) => onChange(e.currentTarget.checked)}
            />
        </div>
    );
}
