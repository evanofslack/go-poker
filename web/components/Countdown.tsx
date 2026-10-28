import { useEffect, useState } from "react";
import classNames from "classnames";

// Countdown shows the time left to act. It counts down locally from the server's
// turnEndsIn, so client and server clocks don't need to agree. total is the
// table's turn time.
export default function Countdown({ endsIn, total }: { endsIn: number; total: number }) {
    const [left, setLeft] = useState(endsIn);

    useEffect(() => {
        const deadline = Date.now() + endsIn;
        setLeft(endsIn);
        const id = setInterval(() => setLeft(Math.max(0, deadline - Date.now())), 200);
        return () => clearInterval(id);
    }, [endsIn]);

    // inside the bottom of the seat plate, so it never spills past the seat
    return (
        <div className="absolute inset-x-3 bottom-1 h-1 rounded-full bg-muted/40 phone:h-1.5">
            <div
                className={classNames(
                    "h-full rounded-full",
                    left < 10000 ? "bg-danger" : "bg-accent"
                )}
                style={{ width: `${Math.min(100, (left / total) * 100)}%` }}
            />
        </div>
    );
}
