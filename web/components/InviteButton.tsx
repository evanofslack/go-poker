import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { inviteLink } from "../lib/invite";
import Button from "./Button";

export default function InviteButton({ compact = false }: { compact?: boolean }) {
    const { appState, dispatch } = useContext(AppContext);
    if (!appState.table) return null;

    const copy = async () => {
        const link = inviteLink(appState.table!);
        try {
            await navigator.clipboard.writeText(link);
            dispatch({ type: "addToast", payload: { message: "invite link copied", kind: "info" } });
        } catch {
            // clipboard needs a secure context; show the link to copy by hand
            dispatch({ type: "addToast", payload: { message: link, kind: "info" } });
        }
    };

    return (
        <Button
            variant={compact ? "ghost" : "secondary"}
            className="px-2 py-0.5 text-sm"
            onClick={copy}
        >
            {compact ? "invite" : "copy invite link"}
        </Button>
    );
}
