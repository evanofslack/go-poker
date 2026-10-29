import { useEffect, useState } from "react";
import { adminGet } from "../lib/admin";

// useAdmin loads path from the admin API, and again every interval ms when
// given. A null path loads nothing.
export function useAdmin<T>(path: string | null, interval?: number) {
    const [data, setData] = useState<T | null>(null);
    const [error, setError] = useState("");

    useEffect(() => {
        setData(null);
        setError("");
        if (!path) return;
        let stopped = false;
        const load = () =>
            adminGet<T>(path).then(
                (d) => {
                    if (stopped) return;
                    setData(d);
                    setError("");
                },
                (e: Error) => !stopped && setError(e.message)
            );
        load();
        const id = interval ? setInterval(load, interval) : undefined;
        return () => {
            stopped = true;
            clearInterval(id);
        };
    }, [path, interval]);

    return { data, error };
}
