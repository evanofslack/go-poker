import { ReactNode, useEffect, useId, useRef } from "react";
import { createPortal } from "react-dom";
import classNames from "classnames";

type dialogProps = {
    title: string;
    onClose: () => void;
    children: ReactNode;
    size?: "sm" | "md" | "lg"; // md and lg are for forms; they scroll when taller than the screen
};

// Dialog is a modal over the page. It's a portal, since a seat's transform would
// otherwise contain the fixed overlay. Escape or a click outside closes it.
export default function Dialog({ title, onClose, children, size = "sm" }: dialogProps) {
    const id = useId();
    const ref = useRef<HTMLDivElement>(null);
    // callers pass onClose inline; a ref keeps game updates from re-running the effect
    const close = useRef(onClose);
    close.current = onClose;

    useEffect(() => {
        const opener = document.activeElement as HTMLElement | null;
        // focus the first control, unless one already took focus
        if (!ref.current?.contains(document.activeElement)) {
            ref.current?.querySelector<HTMLElement>("input, button")?.focus();
        }
        const key = (e: KeyboardEvent) => e.key === "Escape" && close.current();
        document.addEventListener("keydown", key);
        return () => {
            document.removeEventListener("keydown", key);
            opener?.focus();
        };
    }, []);

    return createPortal(
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
            onClick={(e) => e.target === e.currentTarget && onClose()}
        >
            <div
                ref={ref}
                role="dialog"
                aria-modal="true"
                aria-labelledby={id}
                className={classNames(
                    "rounded-xl bg-panel p-4 text-fg shadow-xl",
                    size === "sm" ? "w-72" : "max-h-full w-full overflow-y-auto",
                    size === "md" && "max-w-sm",
                    size === "lg" && "max-w-2xl"
                )}
            >
                <p id={id} className="mb-2 font-semibold">
                    {title}
                </p>
                {children}
            </div>
        </div>,
        document.body
    );
}
