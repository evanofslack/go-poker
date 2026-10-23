import { Message } from "../../interfaces";
import { clock } from "../../lib/time";

export default function ChatMessage({ name, message, time }: Message) {
    if (name == "system" && message.includes("has joined")) {
        return (
            <div className="flex flex-row">
                <p className="text-subtle">[{clock(time)}] &nbsp;</p>
                <p className="italic text-muted">{message}</p>
            </div>
        );
    }

    return (
        <div className="flex flex-row text-muted">
            <p className="text-subtle">[{clock(time)}] &nbsp;</p>
            <p className="font-semibold">{name}: &nbsp; </p>
            <p>{message}</p>
        </div>
    );
}
