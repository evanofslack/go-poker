import { Message } from "../interfaces/index";

export default function ChatMessage({ name, message, timestamp }: Message) {
    if (name == "system" && message.includes("has joined")) {
        return (
            <div className="flex flex-row">
                <p className="text-neutral-500">[{timestamp}] &nbsp;</p>
                <p className="italic text-neutral-400">{message}</p>
            </div>
        );
    }

    return (
        <div className="flex flex-row">
            <p className="text-neutral-400">[{timestamp}] &nbsp;</p>
            <p className="font-semibold">{name}: &nbsp; </p>
            <p>{message}</p>
        </div>
    );
}
