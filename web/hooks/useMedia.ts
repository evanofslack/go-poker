import { useCallback, useSyncExternalStore } from "react";

// phone is a phone held upright, which gets its own seat layout. Keep it in step with
// the phone variant in styles/index.css.
export const phone = "(orientation: portrait) and (max-width: 639px)";

// tall is a screen much taller than wide, like a phone or tablet held upright, where seats
// go down the sides of the table instead of around a ring, and the betting controls sit
// in a taller footer. Keep it in step with the tall variant in styles/index.css.
export const tall = "(max-aspect-ratio: 3/4)";

// useMedia says whether query matches, and re-renders when that changes
export function useMedia(query: string) {
    const subscribe = useCallback(
        (change: () => void) => {
            const m = matchMedia(query);
            m.addEventListener("change", change);
            return () => m.removeEventListener("change", change);
        },
        [query]
    );
    return useSyncExternalStore(
        subscribe,
        () => matchMedia(query).matches,
        () => false
    );
}
