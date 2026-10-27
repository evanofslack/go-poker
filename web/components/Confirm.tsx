import { Button } from "@mantine/core";
import Dialog from "./Dialog";

type confirmProps = {
    title: string;
    message: string;
    action: string;
    onConfirm: () => void;
    onCancel: () => void;
};

// Confirm asks before doing something that can't be undone
export default function Confirm({ title, message, action, onConfirm, onCancel }: confirmProps) {
    return (
        <Dialog title={title} onClose={onCancel}>
            <p className="mb-4 text-sm text-muted">{message}</p>
            <div className="flex justify-end gap-2">
                <Button variant="subtle" onClick={onCancel}>
                    cancel
                </Button>
                <Button variant="filled" color="red" onClick={onConfirm}>
                    {action}
                </Button>
            </div>
        </Dialog>
    );
}
