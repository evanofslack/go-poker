// clock writes a unix ms time as the local time of day, like "14:05"
export function clock(ms: number) {
    return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}

// stamp writes a unix ms time as the local date and time to the second, like "Oct 29 14:05:09"
export function stamp(ms: number) {
    const d = new Date(ms);
    const day = d.toLocaleDateString([], { month: "short", day: "numeric" });
    return `${day} ${d.toLocaleTimeString([], { hour12: false })}`;
}
