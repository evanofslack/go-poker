import { useEffect, useState } from "react";

// useSecondsLeft counts down locally from the server's endsIn, in whole seconds
export function useSecondsLeft(endsIn: number) {
    const [left, setLeft] = useState(endsIn);
    useEffect(() => {
        const deadline = Date.now() + endsIn;
        setLeft(endsIn);
        const id = setInterval(() => setLeft(Math.max(0, deadline - Date.now())), 200);
        return () => clearInterval(id);
    }, [endsIn]);
    return Math.ceil(left / 1000);
}
