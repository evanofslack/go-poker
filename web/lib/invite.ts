import type { Dispatch } from "react";

// inviteLink is a link that opens the join form with the table filled in. A
// password goes in the fragment, which browsers never send to the server.
export function inviteLink(table: string, password?: string | null) {
    const link = `${window.location.origin}${window.location.pathname}?table=${encodeURIComponent(table)}`;
    return password ? `${link}#pw=${encodeURIComponent(password)}` : link;
}

// copyInvite copies the invite link, or shows it to copy by hand where the
// clipboard isn't available (it needs a secure context)
export async function copyInvite(table: string, password: string | null, dispatch: Dispatch<any>) {
    const link = inviteLink(table, password);
    try {
        await navigator.clipboard.writeText(link);
        dispatch({ type: "addToast", payload: { message: "invite link copied", kind: "info" } });
    } catch {
        dispatch({ type: "addToast", payload: { message: link, kind: "info" } });
    }
}

// invitedPassword is the password from an invite link. It takes it out of the
// address bar, so it isn't left on screen or in history.
export function invitedPassword() {
    if (typeof window === "undefined") return "";
    const password = new URLSearchParams(window.location.hash.slice(1)).get("pw") ?? "";
    if (window.location.hash) {
        window.history.replaceState(null, "", window.location.pathname + window.location.search);
    }
    return password;
}

// invitedTable is the table from an invite link, if the page was opened with one
export function invitedTable() {
    if (typeof window === "undefined") return "";
    return new URLSearchParams(window.location.search).get("table") ?? "";
}
