import { Message } from "../interfaces/index";

export default function ChatMessage({ name, message, timestamp }: Message) {
    return (
        <div className="flex flex-row">
            <p className="font-semibold">{name}: </p>
            <p>{message}</p>
            {/* <p>{timestamp}</p> */}
        </div>
    );
}
