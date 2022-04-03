import { MouseEventHandler } from "react";
import classNames from "classnames";

type buttonProps = {
    action: MouseEventHandler<HTMLButtonElement>;
    title: string;
    disabled: boolean;
};

const getAction = (title: string, disabled: boolean) => {
    return classNames(
        {
            // betting over and not winner
            "text-red-600 border-red-600 ": title === "fold",
            "text-green-500 border-green-500 ": title !== "fold",
            "opacity-20": disabled,
        },

        "mx-1 rounded-lg border border-2 px-4 py-3 text-2xl font-semibold"
    );
};

export default function InputButton({ action, title, disabled }: buttonProps) {
    if (disabled) {
        return <div className={getAction(title, disabled)}>{title}</div>;
    }
    return (
        <button className={getAction(title, disabled)} onClick={action}>
            {title}
        </button>
    );
}
