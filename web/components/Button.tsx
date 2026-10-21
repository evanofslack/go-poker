import { ButtonHTMLAttributes } from "react";
import classNames from "classnames";

type buttonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "secondary" | "danger" | "ghost";
};

const variants = {
    primary: "border-accent bg-accent font-semibold text-zinc-950 hover:border-cyan-400 hover:bg-cyan-400",
    secondary: "border-zinc-600 text-fg hover:bg-raised",
    danger: "border-danger text-danger font-semibold hover:bg-danger/10",
    ghost: "border-transparent text-subtle hover:text-fg",
};

export default function Button({ variant = "secondary", className, ...props }: buttonProps) {
    return (
        <button
            className={classNames(
                "rounded-md border-2 px-4 py-2 disabled:cursor-not-allowed disabled:opacity-30",
                variants[variant],
                className
            )}
            {...props}
        />
    );
}
