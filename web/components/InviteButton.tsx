import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { copyInvite } from "../lib/invite";
import Button from "./Button";

export default function InviteButton() {
    const { appState, dispatch } = useContext(AppContext);
    if (!appState.table) return null;

    return (
        <Button
            className="px-2 py-0.5 text-sm"
            onClick={() => copyInvite(appState.table!, appState.password, dispatch)}
        >
            copy invite link
        </Button>
    );
}
