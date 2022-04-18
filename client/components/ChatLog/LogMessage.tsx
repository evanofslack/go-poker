import { Log } from "../../interfaces";

export default function LogMessage({ message, timestamp }: Log) {
    return (
        <div className="flex flex-row">
            <p className="text-neutral-500">[{timestamp}] &nbsp;</p>
            <p className="text-neutral-400">{message}</p>
        </div>
    );
}
