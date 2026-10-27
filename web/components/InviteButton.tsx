import { useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { copyInvite } from "../lib/invite";
import { Button } from "@mantine/core";

export default function InviteButton() {
    const { appState, dispatch } = useContext(AppContext);
    if (!appState.table) return null;

    return (
        <Button size="xs" onClick={() => copyInvite(appState.table!, appState.password, dispatch)}
        >
            copy invite link
        </Button>
    );
}
