import { useContext, useState } from "react";
import { adjustChips, hostAction, HostAction } from "../../actions/actions";
import { useSocket } from "../../hooks/useSocket";
import { PlayerLine } from "../../lib/ledger";
import { AppContext } from "../../providers/AppStore";
import Button from "../Button";
import Dialog from "../Dialog";
import Segmented from "../Segmented";

const input = "w-full rounded-md bg-raised px-2 py-1 text-fg focus:outline-hidden";

// ManagePlayer holds the host's actions on one player. Each closes the dialog;
// the result shows in the panel and the log.
export default function ManagePlayer({ line, onClose }: { line: PlayerLine; onClose: () => void }) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [removing, setRemoving] = useState(false);
    const [mode, setMode] = useState<"add" | "take">("add");
    const [amount, setAmount] = useState("");
    const [reason, setReason] = useState("");
    const game = appState.view;
    if (!socket || !game) return null;

    const { name, status, seat, bot } = line;
    const self = name === appState.username;
    const here = status !== "left" && status !== "disconnected";
    const player = game.players.find((p) => p.name === name);
    const chips = parseInt(amount);

    const act = (action: HostAction) => {
        hostAction(socket, action, name);
        onClose();
    };
    const actions: [string, HostAction, boolean][] = [
        ["make host", "make-host", here && !self && !bot],
        ["make banker", "set-banker", game.ledger.banker !== name && !bot],
        ["sit out", "sit-out-player", status === "playing"],
        ["move to watching", "move-to-spectator", seat !== null && !bot],
    ];

    if (removing) {
        return (
            // its own key, so it mounts fresh and takes focus
            <Dialog key="remove" title={`remove ${name}?`} onClose={onClose}>
                <p className="mb-4 text-sm text-muted">
                    they leave the table and cash out. they can join again.
                </p>
                <div className="flex justify-end gap-2">
                    <Button variant="ghost" className="px-2 py-1 text-sm" onClick={onClose}>
                        cancel
                    </Button>
                    <Button variant="danger" className="px-3 py-1 text-sm" onClick={() => act("remove-player")}>
                        remove
                    </Button>
                </div>
            </Dialog>
        );
    }

    return (
        <Dialog title={`manage ${name}`} onClose={onClose}>
            <div className="flex flex-col gap-1">
                {actions
                    .filter(([, , show]) => show)
                    .map(([label, action]) => (
                        <Button key={action} className="px-3 py-1 text-left text-sm" onClick={() => act(action)}>
                            {label}
                        </Button>
                    ))}
                {!self && status !== "left" && (
                    <Button variant="danger" className="px-3 py-1 text-left text-sm" onClick={() => setRemoving(true)}>
                        remove from table
                    </Button>
                )}
            </div>
            {player && (
                <form
                    className="mt-4 flex flex-col gap-2 border-t border-raised pt-3"
                    onSubmit={(e) => {
                        e.preventDefault();
                        adjustChips(socket, name, mode === "add" ? chips : -chips, reason.trim());
                        onClose();
                    }}
                >
                    <Segmented
                        legend={`chips (stack ${player.stack})`}
                        name="chipMode"
                        options={[
                            { label: "add", value: "add" },
                            { label: "take", value: "take" },
                        ]}
                        value={mode}
                        onChange={setMode}
                    />
                    <input
                        className={input}
                        id="chipAmount"
                        inputMode="numeric"
                        placeholder="amount"
                        aria-label="amount"
                        value={amount}
                        onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
                    />
                    <input
                        className={input}
                        id="chipReason"
                        maxLength={60}
                        placeholder="reason (optional)"
                        aria-label="reason"
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                    />
                    {player.in && <p className="text-xs text-subtle">chips can change once this hand ends</p>}
                    <Button
                        type="submit"
                        variant="primary"
                        className="px-3 py-1 text-sm"
                        disabled={!(chips > 0) || player.in}
                    >
                        {mode === "add" ? "add chips" : "take chips"}
                    </Button>
                </form>
            )}
        </Dialog>
    );
}
