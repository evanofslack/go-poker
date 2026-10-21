import { useContext, useEffect } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { Toast } from "../interfaces/index";

const toastMs = 4000;

function ToastItem({ toast }: { toast: Toast }) {
    const { dispatch } = useContext(AppContext);
    useEffect(() => {
        const id = setTimeout(() => dispatch({ type: "removeToast", payload: toast.id }), toastMs);
        return () => clearTimeout(id);
    }, [toast.id, dispatch]);

    return (
        <button
            className={classNames(
                "pointer-events-auto rounded-md border px-4 py-2 text-sm shadow-xl",
                toast.kind === "error"
                    ? "border-danger bg-panel text-rose-300"
                    : "border-zinc-600 bg-panel text-fg"
            )}
            role={toast.kind === "error" ? "alert" : "status"}
            onClick={() => dispatch({ type: "removeToast", payload: toast.id })}
        >
            {toast.message}
        </button>
    );
}

// Toaster shows short-lived messages, like a rejected action, at the top of the screen
export default function Toaster() {
    const { appState } = useContext(AppContext);
    return (
        <div className="pointer-events-none fixed inset-x-0 top-3 z-50 flex flex-col items-center gap-2">
            {appState.toasts.map((t) => (
                <ToastItem key={t.id} toast={t} />
            ))}
        </div>
    );
}
