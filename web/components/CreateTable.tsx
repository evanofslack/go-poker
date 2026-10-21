import { useContext, useState } from "react";
import { createTable } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import {
    blinds,
    defaultForm,
    handDelays,
    maxPasswordLen,
    seatCounts,
    settingsError,
    SettingsForm,
    stakes,
    toSettings,
    turnTimes,
} from "../lib/settings";
import Button from "./Button";
import Dialog from "./Dialog";
import Segmented from "./Segmented";

type createTableProps = {
    tablename: string;
    username: string;
    onClose: () => void;
};

const input = "w-full rounded-md bg-raised px-2 py-1 text-fg focus:outline-hidden";
const label = "mb-1 block text-xs text-muted";
const digits = (s: string) => s.replace(/\D/g, "");

// CreateTable picks the new table's settings, which are fixed once it exists.
// The table shows once the server confirms with joined-table, which closes this.
export default function CreateTable({ tablename, username, onClose }: createTableProps) {
    const socket = useSocket();
    const { appState } = useContext(AppContext);
    const [form, setForm] = useState<SettingsForm>(defaultForm);
    const [password, setPassword] = useState("");
    const set = (change: Partial<SettingsForm>) => setForm({ ...form, ...change });

    const invalid = settingsError(form);
    const bb = blinds(form)[1];
    const chips = (n: string) => (bb > 0 && parseInt(n) > 0 ? `${parseInt(n) * bb} chips` : "");

    return (
        <Dialog title={`create table ${tablename}`} onClose={onClose} wide>
            <form
                className="flex flex-col gap-4"
                onSubmit={(e) => {
                    e.preventDefault();
                    if (socket && !invalid) {
                        createTable(socket, tablename, username, toSettings(form), password || undefined);
                    }
                }}
            >
                <div>
                    <Segmented
                        legend="stakes"
                        name="stakes"
                        options={[...stakes, "custom"].map((s) => ({ label: s, value: s }))}
                        value={form.stakes}
                        onChange={(s) => set({ stakes: s })}
                    />
                    {form.stakes === "custom" && (
                        <div className="mt-2 flex gap-2">
                            <div className="flex-1">
                                <label className={label} htmlFor="sb">
                                    small blind
                                </label>
                                <input
                                    className={input}
                                    id="sb"
                                    inputMode="numeric"
                                    value={form.sb}
                                    onChange={(e) => set({ sb: digits(e.target.value) })}
                                />
                            </div>
                            <div className="flex-1">
                                <label className={label} htmlFor="bb">
                                    big blind
                                </label>
                                <input
                                    className={input}
                                    id="bb"
                                    inputMode="numeric"
                                    value={form.bb}
                                    onChange={(e) => set({ bb: digits(e.target.value) })}
                                />
                            </div>
                        </div>
                    )}
                </div>
                <div className="flex gap-2">
                    <div className="flex-1">
                        <label className={label} htmlFor="minBuy">
                            min buy-in (big blinds)
                        </label>
                        <input
                            className={input}
                            id="minBuy"
                            inputMode="numeric"
                            value={form.minBuyBB}
                            onChange={(e) => set({ minBuyBB: digits(e.target.value) })}
                        />
                        <span className="text-xs text-subtle">{chips(form.minBuyBB)}</span>
                    </div>
                    <div className="flex-1">
                        <label className={label} htmlFor="maxBuy">
                            max buy-in (big blinds)
                        </label>
                        <input
                            className={input}
                            id="maxBuy"
                            inputMode="numeric"
                            value={form.maxBuyBB}
                            onChange={(e) => set({ maxBuyBB: digits(e.target.value) })}
                        />
                        <span className="text-xs text-subtle">{chips(form.maxBuyBB)}</span>
                    </div>
                </div>
                <div>
                    <label className={label} htmlFor="tablePassword">
                        password (optional, shared in invite links)
                    </label>
                    <input
                        className={input}
                        id="tablePassword"
                        type="text"
                        autoComplete="off"
                        maxLength={maxPasswordLen}
                        value={password}
                        placeholder="none"
                        onChange={(e) => setPassword(e.target.value)}
                    />
                </div>
                <details className="group">
                    <summary className="cursor-pointer text-sm text-muted hover:text-fg">more settings</summary>
                    <div className="mt-3 flex flex-col gap-4">
                        <Segmented
                            legend="seats"
                            name="seats"
                            options={seatCounts.map((n) => ({ label: String(n), value: n }))}
                            value={form.seats}
                            onChange={(n) => set({ seats: n })}
                        />
                        <Segmented
                            legend="turn timer"
                            name="turnTime"
                            options={turnTimes.map((ms) => ({ label: ms ? `${ms / 1000}s` : "off", value: ms }))}
                            value={form.turnTime}
                            onChange={(ms) => set({ turnTime: ms })}
                        />
                        <Segmented
                            legend="pause between hands"
                            name="nextHandDelay"
                            options={handDelays.map((ms) => ({ label: `${ms / 1000}s`, value: ms }))}
                            value={form.nextHandDelay}
                            onChange={(ms) => set({ nextHandDelay: ms })}
                        />
                        <label className="flex items-center gap-2 text-sm">
                            <input
                                type="checkbox"
                                className="accent-cyan-500"
                                checked={form.public}
                                onChange={(e) => set({ public: e.target.checked })}
                            />
                            public <span className="text-xs text-subtle">(not listed anywhere yet)</span>
                        </label>
                        <label className="flex items-center gap-2 text-sm">
                            <input
                                type="checkbox"
                                className="accent-cyan-500"
                                id="approveRebuys"
                                checked={form.approveRebuys}
                                onChange={(e) => set({ approveRebuys: e.target.checked })}
                            />
                            approve rebuys{" "}
                            <span className="text-xs text-subtle">(you ok each rebuy and return to the table)</span>
                        </label>
                    </div>
                </details>
                {(invalid || appState.error) && (
                    <p className="text-sm text-rose-400" role="alert">
                        {invalid || appState.error}
                    </p>
                )}
                <div className="flex justify-end gap-2">
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
