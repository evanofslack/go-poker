import { Message } from "../interfaces/index";

export default function ChatMessage({ name, message, timestamp }: Message) {
    if (name == "system") {
        return (
            <div className="flex flex-row">
                <p className="text-gray-400">[{timestamp}] &nbsp;</p>
                <p className="italic text-gray-400">{message}</p>
            </div>
        );
    }

    return (
        <div className="flex flex-row">
            <p className="text-gray-400">[{timestamp}] &nbsp;</p>
            <p className="font-semibold">{name}: &nbsp; </p>
            <p>{message}</p>
        </div>
    );
}
