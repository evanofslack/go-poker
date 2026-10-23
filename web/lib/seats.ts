import { CSSProperties } from "react";

// A point places a seat on the table, from -1 to 1 across and down, with 0, 0 at the centre
export type Point = { x: number; y: number };

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

// ringPoint places a seat on the ring around a wide table
export function ringPoint(angle: number): Point {
    return { x: bend(Math.cos(angle)), y: bend(Math.sin(angle)) };
}

// tallSlots lay out a tall table, like a phone held upright, for each number of seats.
// A ring there piles seats into the corners, so instead they run down both sides with
// one or two across the top, clear of a band across the middle for the board. They go
// clockwise from your seat at the bottom: up the left, across the top, down the right.
const tallSlots: Record<number, [number, number][]> = {
    2: [[0, 1], [0, -1]],
    3: [[0, 1], [-1, -0.45], [1, -0.45]],
    4: [[0, 1], [-1, 0.45], [0, -1], [1, 0.45]],
    5: [[0, 1], [-1, 0.45], [-0.5, -1], [0.5, -1], [1, 0.45]],
    6: [[0, 1], [-1, 0.45], [-1, -0.45], [0, -1], [1, -0.45], [1, 0.45]],
    7: [[0, 1], [-1, 0.45], [-1, -0.45], [-0.5, -1], [0.5, -1], [1, -0.45], [1, 0.45]],
    8: [[0, 1], [-1, 0.8], [-1, 0.38], [-1, -0.4], [0, -1], [1, -0.4], [1, 0.38], [1, 0.8]],
    9: [[0, 1], [-1, 0.8], [-1, 0.38], [-1, -0.4], [-0.5, -1], [0.5, -1], [1, -0.4], [1, 0.38], [1, 0.8]],
};

// tallPoint places a seat on a tall table, turned so the viewer's own seat is at the bottom
export function tallPoint(seat: number, seats: number, bottom: number): Point {
    const [x, y] = tallSlots[seats][(((seat - bottom) % seats) + seats) % seats];
    return { x, y };
}

// markerAt places a seat's dealer button, blind and bet between the seat and the centre:
// 60% of the way out on a big table, but always far enough in to clear the seat and its
// cards on a narrow one, where 60% would put them under a side seat's cards.
export function markerAt(p: Point): CSSProperties {
    return {
        left: `calc(50% + ${p.x.toFixed(4)} * min(0.6 * (50% - var(--seat-hw)), 50% - 2 * var(--seat-hw) - 2.75rem))`,
        top: `calc(50% + ${p.y.toFixed(4)} * min(0.6 * (50% - var(--seat-hh)), 50% - 2 * var(--seat-hh) - 1.25rem))`,
    };
}

// at positions a seat. The seats stay inside the table area: --seat-hw and --seat-hh are
// half a seat's size.
export function at(p: Point): CSSProperties {
    return {
        left: `calc(50% + ${p.x.toFixed(4)} * (50% - var(--seat-hw)))`,
        top: `calc(50% + ${p.y.toFixed(4)} * (50% - var(--seat-hh)))`,
    };
}

// besideSeat places a seat's bet just off the seat, on the side toward the table's centre.
// It's relative to the seat, so on a phone, where seats crowd the table, the bet stays
// with its own seat.
export function besideSeat(p: Point): CSSProperties {
    // out to the edge of a square, so bets off a seat's corner clear its corner too
    const m = Math.max(Math.abs(p.x), Math.abs(p.y));
    return {
        left: `calc(50% + ${(-p.x / m).toFixed(4)} * (50% + 1.75rem))`,
        top: `calc(50% + ${(-p.y / m).toFixed(4)} * (50% + 1rem))`,
    };
}
