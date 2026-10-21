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
