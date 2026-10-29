import { ReactNode } from "react";
import classNames from "classnames";
import Hint from "./Hint";

type sectionProps = {
    title: string;
    hint?: string;
    className?: string;
    children: ReactNode;
};

// Section groups related controls under a heading, with a rule between it and
// the section before
export default function Section({ title, hint, className, children }: sectionProps) {
    return (
        <section
            aria-label={title}
            className={classNames("flex flex-col gap-3 border-line not-first:border-t not-first:pt-5", className)}
        >
            <h3 className="flex items-center gap-1 text-sm font-semibold text-fg">
                {title}
                {hint && <Hint about={title} text={hint} />}
            </h3>
            {children}
        </section>
    );
}
