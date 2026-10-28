import { ReactNode } from "react";
import { Modal } from "@mantine/core";

type dialogProps = {
    title: string;
    onClose: () => void;
    children: ReactNode;
    size?: "sm" | "md" | "lg"; // md and lg are for forms; they scroll when taller than the screen
};

const widths = { sm: "18rem", md: "24rem", lg: "42rem" };

// Dialog is a modal over the page. Escape or a click outside closes it, and focus stays in
// it while it's open. It's open while it's rendered; callers show and hide it.
// No close button, so focus starts on the first control; no transition, so it's gone at once.
export default function Dialog({ title, onClose, children, size = "sm" }: dialogProps) {
    return (
        <Modal
            opened
            onClose={onClose}
            title={title}
            size={widths[size]}
            centered
            withCloseButton={false}
            transitionProps={{ duration: 0 }}
            classNames={{ header: "min-h-0 pb-2", title: "font-semibold" }}
        >
            {children}
        </Modal>
    );
}
