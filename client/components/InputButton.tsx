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
            "text-rose-600 border-rose-600 font-semibold": title === "fold" || title === "close",
            "text-emerald-500 border-emerald-500 font-normal ":
                title !== "fold" && title !== "close",
            "opacity-20 ": disabled,
        },

        "mx-1 rounded-sm border-2 px-4 py-2 text-xl"
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
