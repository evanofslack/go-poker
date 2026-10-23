import { Log } from "../../interfaces";
import { clock } from "../../lib/time";

export default function LogMessage({ message, time }: Log) {
    return (
        <div className="flex flex-row">
            <p className="text-subtle">[{clock(time)}] &nbsp;</p>
            <p className="text-muted">{message}</p>
        </div>
    );
}
