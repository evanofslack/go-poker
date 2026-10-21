import { RefObject, useEffect } from "react";

// useDismiss calls close on a click outside ref, or on Escape, while open
export function useDismiss(ref: RefObject<HTMLElement | null>, open: boolean, close: () => void) {
    useEffect(() => {
        if (!open) return;
        const click = (e: PointerEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) close();
        };
        const key = (e: KeyboardEvent) => e.key === "Escape" && close();
        document.addEventListener("pointerdown", click);
        document.addEventListener("keydown", key);
        return () => {
            document.removeEventListener("pointerdown", click);
            document.removeEventListener("keydown", key);
        };
    }, [ref, open, close]);
}
