import { useContext } from "react";
import { FiPlay } from "react-icons/fi";
import { ActionIcon, Slider } from "@mantine/core";
import { AppContext } from "../../providers/AppStore";
import { Prefs, colors } from "../../lib/prefs";
import Section from "../Section";
import Segmented from "../Segmented";
import Swatches from "../Swatches";
import Toggle from "../Toggle";
import { Group, play, samples } from "../../lib/sound";

// the sound groups with a switch each, and what they're for
const groups: [Exclude<Group, "turn">, string, string][] = [
    ["timer", "turn timer", "a tick each of the last 5 seconds of your turn"],
    ["decisions", "decisions", "when you're asked to straddle or how many times to run it"],
    ["cards", "cards", "dealing, folding and showing"],
    ["chips", "chips", "bets, checks and pots won"],
    ["chat", "chat", "others' messages"],
];

// Test plays a group's sound, even when it's off
function Test({ group }: { group: Group }) {
    return (
        <ActionIcon size="sm" title={`play ${group}`} aria-label={`play ${group}`} onClick={() => play(group, samples[group], { force: true })}>
            <FiPlay />
        </ActionIcon>
    );
}

const keys: [string, string][] = [
    ["F", "fold"],
    ["C", "check or call"],
    ["R / B", "bet or raise"],
    ["T", "add time from your time bank"],
    ["↑ ↓", "change a bet by a big blind"],
    ["Enter", "place the bet"],
    ["Esc", "close the bet"],
];

// Options are the viewer's own: how the table looks, what it plays, and their
// keys. They're saved in this browser, apart from the table's settings.
export default function Options() {
    const { appState, dispatch } = useContext(AppContext);
    const prefs = appState.prefs;
    const set = (p: Partial<Prefs>) => dispatch({ type: "setPref", payload: p });

    return (
        <div className="flex h-full flex-col gap-6 overflow-auto p-3 text-sm">
            <Section title="look">
                <Segmented
                    legend="animations"
                    hint="system follows your device's reduce motion setting"
                    name="motion"
                    options={[
                        { label: "system", value: "system" },
                        { label: "on", value: "on" },
                        { label: "off", value: "off" },
                    ]}
                    value={prefs.motion}
                    onChange={(motion) => set({ motion })}
                />
                <Segmented
                    legend="deck"
                    hint="two color makes diamonds red and clubs black"
                    name="deck"
                    options={[
                        { label: "four color", value: "four" },
                        { label: "two color", value: "two" },
                    ]}
                    value={prefs.deck}
                    onChange={(deck) => set({ deck })}
                />
                <Swatches legend="table" name="felt" look="felt" options={colors} value={prefs.felt} onChange={(felt) => set({ felt })} />
                <Swatches legend="card backs" name="back" look="back" options={colors} value={prefs.back} onChange={(back) => set({ back })} />
            </Section>
            <Section title="sound">
                <Toggle
                    id="muted"
                    label="mute everything"
                    hint="the speaker at the top right does the same"
                    checked={prefs.muted}
                    onChange={(muted) => set({ muted })}
                    extra={<span className="w-[22px]" />}
                />
                {/* muted, the rest waits; the test buttons still play */}
                <div className="flex flex-col gap-3">
                    <fieldset>
                        <legend className="mb-1 text-xs text-muted">volume</legend>
                        <Slider
                            aria-label="volume"
                            min={0}
                            max={1}
                            step={0.05}
                            label={(v) => `${Math.round(v * 100)}%`}
                            value={prefs.volume}
                            disabled={prefs.muted}
                            onChange={(volume) => set({ volume })}
                        />
                    </fieldset>
                    <div className="flex items-end gap-1">
                        <Segmented
                            legend="your turn"
                            hint="away plays it only when this tab is in the background"
                            name="turn"
                            options={[
                                { label: "always", value: "always" },
                                { label: "when away", value: "away" },
                                { label: "off", value: "off" },
                            ]}
                            value={prefs.turn}
                            disabled={prefs.muted}
                            onChange={(turn) => set({ turn })}
                        />
                        <Test group="turn" />
                    </div>
                    {groups.map(([group, label, hint]) => (
                        <Toggle
                            key={group}
                            id={`sound-${group}`}
                            label={label}
                            hint={hint}
                            checked={prefs[group]}
                            disabled={prefs.muted}
                            onChange={(on) => set({ [group]: on })}
                            extra={<Test group={group} />}
                        />
                    ))}
                </div>
            </Section>
            {/* only for a mouse and keyboard */}
            <Section title="keys" className="hidden pointer-fine:flex">
                <Toggle
                    id="shortcuts"
                    label="keyboard shortcuts"
                    checked={prefs.shortcuts}
                    onChange={(shortcuts) => set({ shortcuts })}
                />
                <div>
                    <dl aria-label="keyboard shortcuts" className={prefs.shortcuts ? "" : "opacity-50"}>
                        {keys.map(([key, does]) => (
                            <div key={key} className="flex gap-2 border-b border-raised py-1.5 last:border-b-0">
                                <dt>
                                    <kbd className="rounded border border-line px-1.5 font-sans text-fg">{key}</kbd>
                                </dt>
                                <dd className="ml-auto text-right text-subtle">{does}</dd>
                            </div>
                        ))}
                    </dl>
                    <p className="py-1.5 text-subtle">Before your turn, F and C pick an action instead.</p>
                </div>
            </Section>
        </div>
    );
}
