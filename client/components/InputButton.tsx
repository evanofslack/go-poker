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
            "text-red-600 border-red-600 ": title === "fold" || title === "close",
            "text-green-500 border-green-500  ": title !== "fold" && title !== "close",
            "opacity-20": disabled,
        },

        "mx-1 border rounded-sm border-2 px-4 py-2 text-2xl font-normal"
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
