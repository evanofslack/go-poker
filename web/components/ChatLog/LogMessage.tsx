import { Log } from "../../interfaces";

export default function LogMessage({ message, timestamp }: Log) {
    return (
        <div className="flex flex-row">
            <p className="text-subtle">[{timestamp}] &nbsp;</p>
            <p className="text-muted">{message}</p>
        </div>
    );
}
