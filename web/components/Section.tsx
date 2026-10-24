import { ReactNode } from "react";
import classNames from "classnames";
import Hint from "./Hint";

type sectionProps = {
    title: string;
    hint?: string;
    className?: string;
    children: ReactNode;
};

// Section groups related controls under a small heading.
export default function Section({ title, hint, className, children }: sectionProps) {
    return (
        <section aria-label={title} className={classNames("flex flex-col gap-3", className)}>
            <h3 className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-xs font-semibold tracking-wide text-subtle uppercase">
                {title}
                {hint && <Hint about={title} text={hint} />}
            </h3>
            {children}
        </section>
    );
}
