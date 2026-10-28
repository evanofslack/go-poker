import classNames from "classnames";

// Chips draws an amount as a short stack of chips. The chips' color and the
// stack's height grow with the amount, counted in big blinds; the amount itself
// goes beside them.
// The colors are whole class names, so tailwind keeps the theme colors they use.
const colors = [
    "[--chip:var(--color-chip-1)]",
    "[--chip:var(--color-chip-2)]",
    "[--chip:var(--color-chip-3)]",
    "[--chip:var(--color-chip-4)]",
];

export default function Chips({ amount, bb, className }: { amount: number; bb: number; className?: string }) {
    const blinds = amount / Math.max(bb, 1);
    const color = colors[blinds < 1 ? 0 : blinds < 5 ? 1 : blinds < 25 ? 2 : 3];
    const count = Math.min(4, Math.max(1, Math.ceil(Math.log2(blinds + 1))));
    return (
        <span aria-hidden className={classNames("relative inline-block shrink-0", className)}>
            {Array.from({ length: count }, (_, i) => (
                // each chip sits a little above the one under it
                <span
                    key={i}
                    className={classNames("chip absolute inset-0", color)}
                    style={{ transform: `translateY(${-i * 3}px)` }}
                />
            ))}
        </span>
    );
}
