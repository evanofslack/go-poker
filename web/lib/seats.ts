import { CSSProperties } from "react";

// seatAngle places seats evenly around the table, clockwise from the bottom, turned so
// the viewer's own seat is at the bottom. Angles are in radians, with y pointing down.
export function seatAngle(seat: number, seats: number, bottom: number) {
    return Math.PI / 2 + ((seat - bottom) * 2 * Math.PI) / seats;
}

// squareness bends the seat ring from an ellipse toward a rectangle. On an ellipse, seats
// between the ends and the sides end up too close to both on a long, thin table.
const squareness = 0.5;

function bend(v: number) {
    return Math.sign(v) * Math.abs(v) ** squareness;
}

// markerAt places a seat's dealer button, blind and bet between the seat and the centre:
// 60% of the way out on a big table, but always far enough in to clear the seat and its
// cards on a narrow one, like a phone, where 60% would put them under a side seat's cards.
export function markerAt(angle: number): CSSProperties {
    const x = bend(Math.cos(angle)).toFixed(4);
    const y = bend(Math.sin(angle)).toFixed(4);
    return {
        left: `calc(50% + ${x} * min(0.6 * (50% - var(--seat-hw)), 50% - 2 * var(--seat-hw) - 2.75rem))`,
        top: `calc(50% + ${y} * min(0.6 * (50% - var(--seat-hh)), 50% - 2 * var(--seat-hh) - 1.25rem))`,
    };
}

// at positions an element a fraction of the way from the table's centre to the seat ring.
// The ring stays inside the table area: --seat-hw and --seat-hh are half a seat's size.
export function at(angle: number, fraction = 1): CSSProperties {
    const x = (bend(Math.cos(angle)) * fraction).toFixed(4);
    const y = (bend(Math.sin(angle)) * fraction).toFixed(4);
    return {
        left: `calc(50% + ${x} * (50% - var(--seat-hw)))`,
        top: `calc(50% + ${y} * (50% - var(--seat-hh)))`,
    };
}

// besideSeat places a seat's bet just off the seat, on the side toward the table's centre.
// It's relative to the seat, so on a phone, where seats crowd the table, the bet stays
// with its own seat.
export function besideSeat(angle: number): CSSProperties {
    // out to the edge of a square, so bets off a seat's corner clear its corner too
    const m = Math.max(Math.abs(Math.cos(angle)), Math.abs(Math.sin(angle)));
    const x = (-Math.cos(angle) / m).toFixed(4);
    const y = (-Math.sin(angle) / m).toFixed(4);
    return {
        left: `calc(50% + ${x} * (50% + 1.75rem))`,
        top: `calc(50% + ${y} * (50% + 1rem))`,
    };
}
