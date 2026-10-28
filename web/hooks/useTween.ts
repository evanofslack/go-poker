import { useEffect, useRef, useState } from "react";

// moving reports whether the table animates now: it has settled after a join
// (data-motion) and the viewer hasn't asked for less motion. Matches the motion
// variant in styles/index.css.
function moving() {
    return !!document.querySelector("[data-motion]") && !matchMedia("(prefers-reduced-motion: reduce)").matches;
}

// useTween counts a number to its new value over a moment, so a stack changing
// reads as chips moving. Without table motion it's just the value. A change of
// round, like a new hand, jumps straight to the value.
export function useTween(value: number, round: unknown = null, ms = 400) {
    const [tween, setTween] = useState<number | null>(null);
    // what's on screen, where the next count starts from
    const shown = useRef(value);
    const last = useRef(round);
    const fresh = round !== last.current;
    useEffect(() => {
        const from = shown.current;
        last.current = round;
        if (from === value || fresh || !moving()) {
            shown.current = value;
            setTween(null);
            return;
        }
        const start = performance.now();
        let frame = 0;
        const step = (now: number) => {
            const k = Math.min(1, (now - start) / ms);
            // fast, then settling
            shown.current = Math.round(from + (value - from) * (1 - (1 - k) ** 3));
            setTween(k < 1 ? shown.current : null);
            if (k < 1) frame = requestAnimationFrame(step);
        };
        frame = requestAnimationFrame(step);
        return () => cancelAnimationFrame(frame);
    }, [value, round, ms]);
    if (tween !== null) return tween;
    // the value just changed and the count hasn't started: hold the old one, not
    // flashing the new one first
    return value !== shown.current && !fresh && moving() ? shown.current : value;
}
