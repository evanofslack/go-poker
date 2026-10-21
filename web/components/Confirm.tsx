import Button from "./Button";
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
                <Button variant="ghost" className="px-2 py-1 text-sm" onClick={onCancel}>
                    cancel
                </Button>
                <Button variant="danger" className="px-3 py-1 text-sm" onClick={onConfirm}>
                    {action}
                </Button>
            </div>
        </Dialog>
    );
}
