import { useContext, useState } from "react";
import { createTable } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import { defaultForm, settingsError, SettingsForm, toSettings } from "../lib/settings";
import Button from "./Button";
import Dialog from "./Dialog";
import SettingsFields from "./SettingsFields";

type createTableProps = {
    tablename: string;
    username: string;
    onClose: () => void;
};

const chips = (n: number) => n.toLocaleString("en");

// summary sums up a valid form in a line, to check before creating
function summary(form: SettingsForm, password: string): string {
    const s = toSettings(form);
    const parts = [
        `${chips(s.sb)}/${chips(s.bb)}`,
        `${chips(s.minBuy)}–${chips(s.maxBuy)} buy-in`,
        `${s.seats} seats`,
        s.turnTime ? `${s.turnTime / 1000}s timer` : "no timer",
    ];
    if (s.ante) parts.push(`${chips(s.ante)} ante`);
    if (s.sevenDeuce) parts.push("7-2 game");
    if (password) parts.push("password");
    return parts.join(" · ");
}

// CreateTable picks the new table's settings. The host can change them later,
// all but the seat count.
// The table shows once the server confirms with joined-table, which closes this.
export default function CreateTable({ tablename, username, onClose }: createTableProps) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [form, setForm] = useState<SettingsForm>(defaultForm);
    const [password, setPassword] = useState("");
    const set = (change: Partial<SettingsForm>) => setForm({ ...form, ...change });

    const invalid = settingsError(form);

    return (
        <Dialog title={`create table ${tablename}`} onClose={onClose} size="lg">
            <form
                className="flex flex-col gap-5"
                onSubmit={(e) => {
                    e.preventDefault();
                    if (socket && !invalid) {
                        createTable(socket, tablename, username, toSettings(form), password || undefined);
                    }
                }}
            >
                <SettingsFields form={form} set={set} password={password} setPassword={setPassword} seats />
                {(invalid || appState.error) && (
                    <p className="text-sm text-error" role="alert">
                        {invalid || appState.error}
                    </p>
                )}
                <div className="flex flex-wrap items-center justify-end gap-2 border-t border-raised pt-3">
                    {!invalid && (
                        <p className="mr-auto text-xs text-subtle tabular-nums" aria-label="summary">
                            {summary(form, password)}
                        </p>
                    )}
                    <Button type="button" variant="ghost" className="px-2 py-1 text-sm" onClick={onClose}>
                        cancel
                    </Button>
                    <Button type="submit" variant="primary" className="px-3 py-1 text-sm" disabled={!!invalid}>
                        create table
                    </Button>
                </div>
            </form>
        </Dialog>
    );
}
