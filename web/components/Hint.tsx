import { useId, useState } from "react";
import { ActionIcon } from "@mantine/core";
import { FiInfo } from "react-icons/fi";

type hintProps = {
    about: string; // what the hint explains, for screen readers
    text: string;
};

// Hint is an info button that shows a line of explanation under its label. Put
// it in a flex-wrap row with the label; the text wraps onto its own line.
export default function Hint({ about, text }: hintProps) {
    const id = useId();
    const [open, setOpen] = useState(false);
    return (
        <>
            <ActionIcon
                size="xs"
                radius="xl"
                aria-label={`about ${about}`}
                aria-expanded={open}
                aria-controls={id}
                title={text}
                onClick={() => setOpen(!open)}
            >
                <FiInfo className="size-3.5" />
            </ActionIcon>
            <span
                id={id}
                hidden={!open}
                className="order-last basis-full text-xs font-normal tracking-normal text-subtle normal-case"
            >
                {text}
            </span>
        </>
    );
}
