import { useEffect, useState } from "react";
import classNames from "classnames";

// matches the server's turnTimeout
const turnMs = 30000;

// Countdown shows the time left to act. It counts down locally from the server's
// turnEndsIn, so client and server clocks don't need to agree.
export default function Countdown({ endsIn }: { endsIn: number }) {
    const [left, setLeft] = useState(endsIn);

    useEffect(() => {
        const deadline = Date.now() + endsIn;
        setLeft(endsIn);
        const id = setInterval(() => setLeft(Math.max(0, deadline - Date.now())), 200);
        return () => clearInterval(id);
    }, [endsIn]);

    // inside the bottom of the seat plate, so it never spills past the seat
    return (
        <div className="absolute inset-x-3 bottom-1 h-1 rounded-full bg-zinc-400/40">
            <div
                className={classNames(
                    "h-full rounded-full",
                    left < 10000 ? "bg-danger" : "bg-accent"
                )}
                style={{ width: `${Math.min(100, (left / turnMs) * 100)}%` }}
            />
        </div>
    );
}
