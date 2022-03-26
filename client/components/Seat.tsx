import { useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { takeSeat } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";

export default function Seat() {
    const socket = useSocket();
    const { appState, dispatch } = useContext(AppContext);
    const [open, setOpen] = useState(true);

    const handleClick = () => {
        setOpen(!open);
        if (socket) {
            takeSeat(socket, appState.username, 1000);
        }
    };

    if (open) {
        return <button onClick={handleClick}>Open</button>;
    } else {
        return <div>Taken by {appState.username}</div>;
    }
}
