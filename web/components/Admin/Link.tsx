import { ReactNode } from "react";

// th styles the admin tables' header cells
export const th = "font-normal text-subtle";

// Link is text that opens something else on the admin page: a table, or the
// audit of an address or name
export function Link({ onClick, children }: { onClick: () => void; children: ReactNode }) {
    return (
        <button
            type="button"
            className="cursor-pointer text-accent hover:underline"
            onClick={(e) => {
                e.stopPropagation();
                onClick();
            }}
        >
            {children}
        </button>
    );
}
