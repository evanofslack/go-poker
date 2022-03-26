import Chat from "./Chat";

export default function Game() {
    return (
        <div className="flex h-screen flex-row items-center justify-center">
            <div className="m-24 flex h-1/2 w-3/5 items-center justify-center rounded-3xl bg-green-600">
                <div>Table</div>
            </div>
            <div className="absolute left-0 bottom-0">
                <Chat />
            </div>
        </div>
    );
}
