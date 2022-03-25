import { Message } from "../interfaces/index";

export default function ChatMessage({ name, message }: Message) {
    return (
        <div>
            <p>{name}</p>
            <p>{message}</p>
        </div>
    );
}
