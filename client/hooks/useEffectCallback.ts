import { useEffect, useRef } from "react";

export default function useEffectCallback(callback: CallableFunction) {
    const isFirstRender = useRef(true);
    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        callback();
    }, [callback]);
}
