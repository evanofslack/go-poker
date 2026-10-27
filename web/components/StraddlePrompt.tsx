import { useContext } from "react";
import { straddle } from "../actions/actions";
import { useSecondsLeft } from "../hooks/useSecondsLeft";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import Button from "./Button";

// StraddlePrompt asks the player under the gun, then each re-straddler, whether
// to straddle before the deal. Everyone else sees who's deciding.
export default function StraddlePrompt() {
  const socket = useSocket();
  const { appState } = useContext(AppContext);
  const view = appState.view;
  const left = useSecondsLeft(view?.straddleEndsIn ?? 0);
  const ask = view?.straddleAsk;
  if (
    !socket ||
    !view ||
    !ask ||
    ask.seat === 0 ||
    appState.connection !== "open"
  )
    return null;

  if (ask.seat !== view.you) {
    const name = view.players.find((p) => p.seat === ask.seat)?.name;
    return (
      <p className="self-center text-sm text-subtle" data-straddle-wait>
        {name} is deciding to straddle{" "}
        <span className="tabular-nums">{left}s</span>
      </p>
    );
  }
  return (
    <div
      role="group"
      aria-label="straddle"
      className="flex items-center gap-2"
      data-straddle-ask
    >
      <span className="text-sm text-subtle tabular-nums">{left}s</span>
      <Button
        variant="primary"
        className="px-2 sm:px-4 sm:text-xl"
        onClick={() => straddle(socket, true)}
      >
        straddle {ask.amount}
      </Button>
      <Button
        className="px-2 sm:px-4 sm:text-xl"
        onClick={() => straddle(socket, false)}
      >
        no
      </Button>
    </div>
  );
}
