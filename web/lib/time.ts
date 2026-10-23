// clock writes a unix ms time as the local time of day, like "14:05"
export function clock(ms: number) {
    return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}
