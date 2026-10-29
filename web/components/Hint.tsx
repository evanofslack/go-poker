import { KeyboardEvent, useState } from "react";
import { ActionIcon, Popover } from "@mantine/core";
import { FiInfo } from "react-icons/fi";

type hintProps = {
    about: string; // what the hint explains, for screen readers
    text: string;
};

// Hint is an info button that shows a line of explanation in a popover, so
// nothing around it moves
export default function Hint({ about, text }: hintProps) {
    const [open, setOpen] = useState(false);
    // Escape closes just the hint, not the dialog it's in, which skips keys from
    // elements marked data-mantine-stop-propagation
    const escape = (e: KeyboardEvent) => {
        if (e.key !== "Escape" || !open) return;
        e.stopPropagation();
        setOpen(false);
    };
    const stop = { "data-mantine-stop-propagation": open || undefined };
    return (
        <Popover
            opened={open}
            onChange={setOpen}
            closeOnEscape={false}
            withArrow
            position="bottom-start"
            width={256}
            shadow="md"
            transitionProps={{ duration: 0 }}
        >
            <Popover.Target>
                <ActionIcon
                    size="xs"
                    radius="xl"
                    aria-label={`about ${about}`}
                    onClick={() => setOpen(!open)}
                    onKeyDown={escape}
                    {...stop}
                >
                    <FiInfo className="size-3.5" />
                </ActionIcon>
            </Popover.Target>
            <Popover.Dropdown className="px-3 py-2 text-xs font-normal text-muted" onKeyDown={escape} {...stop}>
                {text}
            </Popover.Dropdown>
        </Popover>
    );
}
