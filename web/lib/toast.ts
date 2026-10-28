import { notifications } from "@mantine/notifications";

// toast shows a short message at the top of the screen, like a copied link or a rejected action
export function toast(message: string, kind: "info" | "error" = "info") {
    notifications.show({ message, color: kind === "error" ? "red" : undefined });
}
