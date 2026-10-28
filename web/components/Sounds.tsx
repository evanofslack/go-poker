import { useContext, useEffect, useRef } from "react";
import { AppContext } from "../providers/AppStore";
import { useSecondsLeft } from "../hooks/useSecondsLeft";
import { play, preload } from "../lib/sound";

// Sounds plays what the view says is happening to you: your turn starting, the
// last seconds on its clock, and being asked to straddle or vote on running it.
// The table's own sounds come with its events (providers/WebSocket.tsx). It
// renders nothing, apart from the game so the countdown doesn't re-render it.
export default function Sounds() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    const myTurn = !!view?.legal.turn;
    const left = useSecondsLeft(myTurn ? view!.turnEndsIn : 0);
    const vote = view?.runVote;
    const asked =
        !!view &&
        (view.straddleAsk?.seat === view.you ||
            (!!vote && vote.seats.includes(view.you) && !(view.you in (vote.picks ?? {}))));
    const was = useRef({ myTurn: false, asked: false });

    useEffect(preload, []);

    useEffect(() => {
        const started = myTurn && !was.current.myTurn;
        was.current.myTurn = myTurn;
        if (!started) return;
        // wait a moment, since a pre-action can take the turn straight away
        const id = setTimeout(() => play("turn", "glass-002"), 300);
        return () => clearTimeout(id);
    }, [myTurn]);

    useEffect(() => {
        if (myTurn && left >= 1 && left <= 5) play("timer", "tick-002");
    }, [myTurn, left]);

    useEffect(() => {
        if (asked && !was.current.asked) play("decisions", "question-001");
        was.current.asked = asked;
    }, [asked]);

    return null;
}
