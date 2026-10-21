// inviteLink is a link that opens the join form with the table filled in
export function inviteLink(table: string) {
    return `${window.location.origin}${window.location.pathname}?table=${encodeURIComponent(table)}`;
}

// invitedTable is the table from an invite link, if the page was opened with one
export function invitedTable() {
    if (typeof window === "undefined") return "";
    return new URLSearchParams(window.location.search).get("table") ?? "";
}
