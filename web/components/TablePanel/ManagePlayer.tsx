import { useContext, useState } from "react";
import { adjustChips, hostAction, HostAction } from "../../actions/actions";
import { useSocket } from "../../hooks/useSocket";
import { PlayerLine } from "../../lib/ledger";
import { AppContext } from "../../providers/AppStore";
import { Button, TextInput } from "@mantine/core";
import Dialog from "../Dialog";
import Segmented from "../Segmented";

// label styles a text input's label like the rest of the form's
const label = { label: "mb-1 text-xs font-normal text-muted" };

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
    // the actions that apply to this player, by what they change
    const groups: [string, [string, HostAction, boolean][]][] = [
        [
            "role",
            [
                ["make host", "make-host", here && !self && !bot],
                ["make banker", "set-banker", game.ledger.banker !== name && !bot],
            ],
        ],
        [
            "seat",
            [
                ["sit out", "sit-out-player", status === "playing"],
                ["move to watching", "move-to-spectator", seat !== null && !bot],
            ],
        ],
    ];

    if (removing) {
        return (
            // its own key, so it mounts fresh and takes focus
            <Dialog key="remove" title={`remove ${name}?`} onClose={onClose}>
                <p className="mb-4 text-sm text-muted">
                    they leave the table and cash out. they can join again.
                </p>
                <div className="flex justify-end gap-2">
                    <Button variant="subtle" onClick={onClose}>
                        cancel
                    </Button>
                    <Button variant="filled" color="red" onClick={() => act("remove-player")}>
                        remove
                    </Button>
                </div>
            </Dialog>
        );
    }

    return (
        <Dialog title={`manage ${name}`} onClose={onClose}>
            <div className="flex flex-col gap-3">
                {groups.map(([group, actions]) => {
                    const shown = actions.filter(([, , show]) => show);
                    if (!shown.length) return null;
                    return (
                        <div key={group} role="group" aria-label={group} className="flex flex-col gap-1">
                            <p className="text-xs text-muted">{group}</p>
                            {shown.map(([label, action]) => (
                                <Button key={action} justify="flex-start" onClick={() => act(action)}>
                                    {label}
                                </Button>
                            ))}
                        </div>
                    );
                })}
                {!self && status !== "left" && (
                    <Button variant="outline" color="red" justify="flex-start" onClick={() => setRemoving(true)}>
                        remove from table
                    </Button>
                )}
            </div>
            {player && (
                <form
                    className="mt-4 flex flex-col gap-3 border-t border-line pt-3"
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
                    <TextInput
                        id="chipAmount"
                        label="amount"
                        classNames={label}
                        inputMode="numeric"
                        value={amount}
                        onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
                    />
                    <TextInput
                        id="chipReason"
                        label="reason"
                        classNames={label}
                        maxLength={60}
                        placeholder="optional"
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                    />
                    {player.in && <p className="text-xs text-subtle">chips can change once this hand ends</p>}
                    <div className="flex justify-end gap-2">
                        <Button type="button" variant="subtle" onClick={onClose}>
                            cancel
                        </Button>
                        <Button type="submit" variant="filled" disabled={!(chips > 0) || player.in}>
                            {mode === "add" ? "add chips" : "take chips"}
                        </Button>
                    </div>
                </form>
            )}
        </Dialog>
    );
}
