import { Tooltip } from "@mantine/core";

// StyleHeader is a play style column's short label, with what it means in a
// tooltip that opens on hover, focus or a tap
export default function StyleHeader({ label, title }: { label: string; title: string }) {
    return (
        <Tooltip label={title} multiline w={220} fz="xs" withArrow events={{ hover: true, focus: true, touch: true }}>
            <span tabIndex={0} className="cursor-help underline decoration-dotted underline-offset-2">
                {label}
            </span>
        </Tooltip>
    );
}
