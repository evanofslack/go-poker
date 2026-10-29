// The admin API, on the go server's admin address. The admin page is served
// from there too, so it's same origin; NEXT_PUBLIC_ADMIN_URL points next dev at it.
const base = process.env.NEXT_PUBLIC_ADMIN_URL || "";

// pollMs is how often the admin page reloads the tables
export const pollMs = 5000;

export async function adminGet<T>(path: string): Promise<T> {
    const res = await fetch(`${base}/admin/api/${path}`);
    if (!res.ok) {
        throw new Error((await res.text()).trim() || res.statusText);
    }
    return res.json();
}

// AuditFilter picks audit entries; empty fields match anything
export type AuditFilter = {
    table?: string; // id or name
    person?: string;
    name?: string;
    ip?: string;
    kind?: string;
    refused?: boolean;
};

export function auditPath(filter: AuditFilter, limit: number, before?: number) {
    const q = new URLSearchParams();
    for (const [key, value] of Object.entries(filter)) {
        if (value) q.set(key, String(value));
    }
    q.set("limit", String(limit));
    if (before) q.set("before", String(before));
    return `audit?${q}`;
}
