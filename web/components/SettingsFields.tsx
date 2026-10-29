import { ReactNode } from "react";
import {
    blinds,
    antes,
    bombPots,
    bombSummary,
    bountyHand,
    decisionTimes,
    decks,
    games,
    limits,
    paceSummary,
    rulesSummary,
    straddleModes,
    sevenDeuces,
    handDelays,
    maxPasswordLen,
    seatCounts,
    SettingsForm,
    stakes,
    timeBanks,
    timeCards,
    timeCardsHeld,
    timeEveries,
    turnTimes,
} from "../lib/settings";
import { Accordion, TextInput } from "@mantine/core";
import Hint from "./Hint";
import Section from "./Section";
import Segmented from "./Segmented";
import SpecialSchedule from "./SpecialSchedule";
import Toggle from "./Toggle";

type settingsFieldsProps = {
    form: SettingsForm;
    set: (change: Partial<SettingsForm>) => void;
    password: string;
    setPassword: (password: string) => void;
    seats: boolean; // the seat count can't change once the table exists
};

const digits = (s: string) => s.replace(/\D/g, "");
const chips = (n: number) => n.toLocaleString("en");

// FieldLabel labels a text input, with an optional hint
function FieldLabel({ htmlFor, hint, children }: { htmlFor: string; hint?: string; children: string }) {
    return (
        <div className="mb-1 flex items-center gap-1 text-xs text-muted">
            <label htmlFor={htmlFor}>{children}</label>
            {hint && <Hint about={children} text={hint} />}
        </div>
    );
}

// More is a section that starts closed, with a line saying what's in it
function More({ title, summary, children }: { title: string; summary: string; children: ReactNode }) {
    return (
        <Accordion.Item value={title} aria-label={title}>
            <Accordion.Control>
                <span className="block text-sm font-semibold text-fg">{title}</span>
                <span className="block text-xs text-subtle">{summary}</span>
            </Accordion.Control>
            <Accordion.Panel>
                <div className="flex flex-col gap-3">{children}</div>
            </Accordion.Panel>
        </Accordion.Item>
    );
}

// SettingsFields edits a table's settings, for a new table or the host changing one.
// The game and who can join are always open; the rest have good defaults, so
// they start closed.
export default function SettingsFields({ form, set, password, setPassword, seats }: settingsFieldsProps) {
    const bb = blinds(form)[1];
    const [min, max] = [parseInt(form.minBuyBB), parseInt(form.maxBuyBB)];
    const buyInChips = bb > 0 && min > 0 && max > 0 ? `${chips(min * bb)}–${chips(max * bb)} chips` : "";

    return (
        <div className="flex flex-col gap-6">
            <Section title="game">
                <div className="flex flex-wrap gap-x-4 gap-y-3">
                    <Segmented
                        legend="game"
                        name="game"
                        options={games}
                        value={form.game}
                        // the 7-2 bounty is a hold'em game
                        onChange={(g) => set({ game: g, sevenDeuce: g === "holdem" ? form.sevenDeuce : 0 })}
                    />
                    <Segmented
                        legend="limit"
                        hint="pot limit caps every bet and raise at the size of the pot"
                        name="limit"
                        options={limits}
                        value={form.limit}
                        onChange={(l) => set({ limit: l })}
                    />
                    <Segmented
                        legend="deck"
                        hint="short deck takes out the 2s to 5s: a flush beats a full house, three of a kind beats a straight, and A-6-7-8-9 is a straight"
                        name="deck"
                        options={decks}
                        value={form.deck}
                        onChange={(d) => set({ deck: d })}
                    />
                </div>
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
                                <FieldLabel htmlFor="sb" hint="0 for none: antes and a big blind, as short deck often plays">
                                    small blind
                                </FieldLabel>
                                <TextInput
                                    id="sb"
                                    inputMode="numeric"
                                    value={form.sb}
                                    onChange={(e) => set({ sb: digits(e.target.value) })}
                                />
                            </div>
                            <div className="flex-1">
                                <FieldLabel htmlFor="bb">big blind</FieldLabel>
                                <TextInput
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
                        <TextInput
                            className="flex-1"
                            id="minBuy"
                            aria-label="min buy-in, in big blinds"
                            inputMode="numeric"
                            value={form.minBuyBB}
                            onChange={(e) => set({ minBuyBB: digits(e.target.value) })}
                        />
                        <span className="text-subtle">–</span>
                        <TextInput
                            className="flex-1"
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
                <div className="flex flex-wrap gap-x-4 gap-y-3">
                    <Segmented
                        legend="ante"
                        hint="a share of the big blind, paid by everyone dealt in"
                        name="ante"
                        options={antes.map((a) => ({ label: a ? `${a * 100}%` : "off", value: a }))}
                        value={form.ante}
                        onChange={(a) => set({ ante: a })}
                    />
                    {seats && (
                        <Segmented
                            legend="seats"
                            name="seats"
                            options={seatCounts.map((n) => ({ label: String(n), value: n }))}
                            value={form.seats}
                            onChange={(n) => set({ seats: n })}
                        />
                    )}
                </div>
            </Section>
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
                    <TextInput
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
            <Accordion
                multiple
                classNames={{
                    root: "border-t border-line",
                    item: "border-line",
                    control: "px-0 py-1 hover:bg-transparent",
                    content: "px-0 pb-5",
                }}
            >
                <More title="pace" summary={paceSummary(form)}>
                    <Segmented
                        legend="turn timer"
                        name="turnTime"
                        options={turnTimes.map((ms) => ({ label: ms ? `${ms / 1000}s` : "off", value: ms }))}
                        value={form.turnTime}
                        onChange={(ms) => set({ turnTime: ms })}
                    />
                    {form.turnTime > 0 && (
                        <Segmented
                            legend="time bank"
                            hint="cards a player sits down with; each adds time to their turn, spent when they ask or when their time runs out"
                            name="timeBank"
                            options={timeBanks.map((n) => ({ label: n ? String(n) : "off", value: n }))}
                            value={form.timeMax ? form.timeBank : 0}
                            onChange={(n) => set({ timeBank: n, timeMax: n ? Math.max(n, timeCardsHeld) : 0 })}
                        />
                    )}
                    {form.turnTime > 0 && form.timeMax > 0 && (
                        <div className="flex flex-wrap gap-x-4 gap-y-3">
                            <Segmented
                                legend="time card"
                                name="timeCard"
                                options={timeCards.map((ms) => ({ label: `${ms / 1000}s`, value: ms }))}
                                value={form.timeCard}
                                onChange={(ms) => set({ timeCard: ms })}
                            />
                            <Segmented
                                legend="earn a card"
                                hint={`every so many hands dealt in, up to ${form.timeMax}`}
                                name="timeEvery"
                                options={timeEveries.map((n) => ({ label: n ? `${n} hands` : "never", value: n }))}
                                value={form.timeEvery}
                                onChange={(n) => set({ timeEvery: n })}
                            />
                        </div>
                    )}
                    <Segmented
                        legend="between hands"
                        name="nextHandDelay"
                        options={handDelays.map((ms) => ({ label: `${ms / 1000}s`, value: ms }))}
                        value={form.nextHandDelay}
                        onChange={(ms) => set({ nextHandDelay: ms })}
                    />
                </More>
                <More title="house rules" summary={rulesSummary(form)}>
                    {form.game === "holdem" && (
                        <Segmented
                            legend={`${bountyHand(form)} bounty`}
                            hint={`win the pot with ${bountyHand(form)} and show it, and everyone dealt in pays you this`}
                            name="sevenDeuce"
                            options={sevenDeuces.map((n) => ({ label: n ? `${n} bb` : "off", value: n }))}
                            value={form.sevenDeuce}
                            onChange={(n) => set({ sevenDeuce: n })}
                        />
                    )}
                    <Segmented
                        legend="straddles"
                        hint="under the gun can post 2 bb before the deal, and each next player double that; forced posts the first one"
                        name="straddle"
                        options={straddleModes.map((m) => ({ label: m, value: m }))}
                        value={form.straddle}
                        onChange={(m) => set({ straddle: m })}
                    />
                    <Toggle
                        id="runIt"
                        label="run it"
                        hint="when everyone left is all in, or all but one, they can run the rest of the board out up to 4 times"
                        checked={form.runIt}
                        onChange={(on) => set({ runIt: on })}
                    />
                    {(form.straddle !== "off" || form.runIt) && (
                        <Segmented
                            legend="to decide"
                            hint="how long a player has to answer whether to straddle, or how many times to run it"
                            name="decisionTime"
                            options={decisionTimes.map((ms) => ({ label: `${ms / 1000}s`, value: ms }))}
                            value={form.decisionTime}
                            onChange={(ms) => set({ decisionTime: ms })}
                        />
                    )}
                    <Toggle
                        id="duckHunt"
                        label="duck hunt"
                        hint="see the rest of the board after you fold"
                        checked={form.duckHunt}
                        onChange={(on) => set({ duckHunt: on })}
                    />
                </More>
                <More title="bomb pots" summary={bombSummary(form)}>
                    <Toggle
                        id="bombPots"
                        label="deal bomb pots"
                        hint="everyone puts in the same amount and the hand starts on the flop. the host can call one any time from the host tab"
                        checked={form.bombPot > 0}
                        onChange={(on) => set({ bombPot: on ? 2 : 0 })}
                    />
                    {form.bombPot > 0 && (
                        <>
                            <Segmented
                                legend="bomb pot"
                                name="bombPot"
                                options={bombPots.map((n) => ({ label: `${n} bb`, value: n }))}
                                value={form.bombPot}
                                onChange={(n) => set({ bombPot: n })}
                            />
                            <SpecialSchedule
                                id="doubleBoard"
                                special={form.special}
                                specialEvery={form.specialEvery}
                                specialUnit={form.specialUnit}
                                onChange={set}
                            />
                        </>
                    )}
                </More>
            </Accordion>
        </div>
    );
}
