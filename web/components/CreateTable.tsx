import { useContext, useState } from "react";
import { createTable } from "../actions/actions";
import { useSocket } from "../hooks/useSocket";
import { AppContext } from "../providers/AppStore";
import {
    blinds,
    defaultForm,
    antes,
    sevenDeuces,
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
import Hint from "./Hint";
import Section from "./Section";
import Segmented from "./Segmented";
import Toggle from "./Toggle";

type createTableProps = {
    tablename: string;
    username: string;
    onClose: () => void;
};

const input = "w-full rounded-md bg-raised px-2 py-1 text-fg focus:outline-hidden";
const label = "mb-1 block text-xs text-muted";
const digits = (s: string) => s.replace(/\D/g, "");
const chips = (n: number) => n.toLocaleString("en");

// FieldLabel labels a text input, with an optional hint
function FieldLabel({ htmlFor, hint, children }: { htmlFor: string; hint?: string; children: string }) {
    return (
        <div className="mb-1 flex flex-wrap items-center gap-x-1 gap-y-0.5 text-xs text-muted">
            <label htmlFor={htmlFor}>{children}</label>
            {hint && <Hint about={children} text={hint} />}
        </div>
    );
}

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
    const [min, max] = [parseInt(form.minBuyBB), parseInt(form.maxBuyBB)];
    const buyInChips = bb > 0 && min > 0 && max > 0 ? `${chips(min * bb)}–${chips(max * bb)} chips` : "";

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
                <div className="grid gap-5 sm:grid-cols-2 sm:gap-x-8">
                    <div className="flex flex-col gap-5">
                        <Section title="game">
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
                            <div>
                                <FieldLabel htmlFor="minBuy" hint="the least and most chips a player can sit down with">
                                    buy-in
                                </FieldLabel>
                                <div className="flex items-center gap-2 text-sm">
                                    <input
                                        className={input}
                                        id="minBuy"
                                        aria-label="min buy-in, in big blinds"
                                        inputMode="numeric"
                                        value={form.minBuyBB}
                                        onChange={(e) => set({ minBuyBB: digits(e.target.value) })}
                                    />
                                    <span className="text-subtle">–</span>
                                    <input
                                        className={input}
                                        id="maxBuy"
                                        aria-label="max buy-in, in big blinds"
                                        inputMode="numeric"
                                        value={form.maxBuyBB}
                                        onChange={(e) => set({ maxBuyBB: digits(e.target.value) })}
                                    />
                                    <span className="text-muted">bb</span>
                                </div>
                                <p className="mt-1 h-4 text-xs text-subtle tabular-nums">{buyInChips}</p>
                            </div>
                            <Segmented
                                legend="ante"
                                hint="a share of the big blind, paid by everyone dealt in"
                                name="ante"
                                options={antes.map((a) => ({ label: a ? `${a * 100}%` : "off", value: a }))}
                                value={form.ante}
                                onChange={(a) => set({ ante: a })}
                            />
                            <Segmented
                                legend="seats"
                                name="seats"
                                options={seatCounts.map((n) => ({ label: String(n), value: n }))}
                                value={form.seats}
                                onChange={(n) => set({ seats: n })}
                            />
                        </Section>
                        <Section title="pace">
                            <Segmented
                                legend="turn timer"
                                name="turnTime"
                                options={turnTimes.map((ms) => ({ label: ms ? `${ms / 1000}s` : "off", value: ms }))}
                                value={form.turnTime}
                                onChange={(ms) => set({ turnTime: ms })}
                            />
                            <Segmented
                                legend="between hands"
                                name="nextHandDelay"
                                options={handDelays.map((ms) => ({ label: `${ms / 1000}s`, value: ms }))}
                                value={form.nextHandDelay}
                                onChange={(ms) => set({ nextHandDelay: ms })}
                            />
                        </Section>
                    </div>
                    <div className="flex flex-col gap-5">
                        <Section title="access">
                            <Toggle
                                id="public"
                                label="public"
                                hint="listed on the home page for anyone to join"
                                checked={form.public}
                                onChange={(on) => set({ public: on })}
                            />
                            <Toggle
                                id="approveRebuys"
                                label="approve rebuys"
                                hint="you approve each rebuy and every return to the table"
                                checked={form.approveRebuys}
                                onChange={(on) => set({ approveRebuys: on })}
                            />
                            <div>
                                <FieldLabel htmlFor="tablePassword" hint="optional; it's shared in invite links">
                                    password
                                </FieldLabel>
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
                        </Section>
                        <Section title="house rules">
                            <Segmented
                                legend="7-2 bounty"
                                hint="win the pot with 7-2 and show it, and everyone dealt in pays you this"
                                name="sevenDeuce"
                                options={sevenDeuces.map((n) => ({ label: n ? `${n} bb` : "off", value: n }))}
                                value={form.sevenDeuce}
                                onChange={(n) => set({ sevenDeuce: n })}
                            />
                            <Toggle
                                id="duckHunt"
                                label="duck hunt"
                                hint="see the rest of the board after you fold"
                                checked={form.duckHunt}
                                onChange={(on) => set({ duckHunt: on })}
                            />
                        </Section>
                    </div>
                </div>
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
