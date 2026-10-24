import { useContext, useState } from "react";
import { setPassword as sendPassword, updateSettings } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { defaultForm, fromSettings, settingsError, SettingsForm, toSettings } from "../lib/settings";
import Button from "./Button";
import Dialog from "./Dialog";
import SettingsFields from "./SettingsFields";

// EditSettings lets the host change the table's settings, all but the seat
// count. Mid-hand they apply once the hand ends.
export default function EditSettings({ onClose }: { onClose: () => void }) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const game = appState.view;
    const [form, setForm] = useState<SettingsForm>(() =>
        game ? fromSettings(game.next ?? game.settings) : defaultForm
    );
    // empty when the table has a password this client doesn't know, which saving leaves alone
    const known = appState.password ?? "";
    const [password, setPassword] = useState(known);
    const set = (change: Partial<SettingsForm>) => setForm({ ...form, ...change });
    if (!socket || !game) return null;

    const invalid = settingsError(form);
    const midHand = game.street !== "";

    return (
        <Dialog title="table settings" onClose={onClose} size="lg">
            <form
                className="flex flex-col gap-5"
                onSubmit={(e) => {
                    e.preventDefault();
                    if (invalid) return;
                    updateSettings(socket, toSettings(form));
                    if (password !== known) {
                        sendPassword(socket, password);
                    }
                    onClose();
                }}
            >
                <SettingsFields form={form} set={set} password={password} setPassword={setPassword} seats={false} />
                {invalid && (
                    <p className="text-sm text-error" role="alert">
                        {invalid}
                    </p>
                )}
                <div className="flex flex-wrap items-center justify-end gap-2 border-t border-raised pt-3">
                    {midHand && <p className="mr-auto text-xs text-subtle">changes apply after this hand</p>}
                    <Button type="button" variant="ghost" className="px-2 py-1 text-sm" onClick={onClose}>
                        cancel
                    </Button>
                    <Button type="submit" variant="primary" className="px-3 py-1 text-sm" disabled={!!invalid}>
                        save
                    </Button>
                </div>
            </form>
        </Dialog>
    );
}
