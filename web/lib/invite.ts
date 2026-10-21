import type { Dispatch } from "react";

// inviteLink is a link that opens the join form with the table filled in
export function inviteLink(table: string) {
    return `${window.location.origin}${window.location.pathname}?table=${encodeURIComponent(table)}`;
}

// copyInvite copies the invite link, or shows it to copy by hand where the
// clipboard isn't available (it needs a secure context)
export async function copyInvite(table: string, dispatch: Dispatch<any>) {
    const link = inviteLink(table);
    try {
        await navigator.clipboard.writeText(link);
        dispatch({ type: "addToast", payload: { message: "invite link copied", kind: "info" } });
    } catch {
        dispatch({ type: "addToast", payload: { message: link, kind: "info" } });
    }
}

// invitedTable is the table from an invite link, if the page was opened with one
export function invitedTable() {
    if (typeof window === "undefined") return "";
    return new URLSearchParams(window.location.search).get("table") ?? "";
}
