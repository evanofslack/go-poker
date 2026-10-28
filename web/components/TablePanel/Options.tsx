import { useContext } from "react";
import { AppContext } from "../../providers/AppStore";
import { Prefs } from "../../lib/prefs";
import Section from "../Section";
import Segmented from "../Segmented";
import Toggle from "../Toggle";

const keys: [string, string][] = [
    ["F", "fold"],
    ["C", "check or call"],
    ["R / B", "bet or raise"],
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
        <div className="flex h-full flex-col gap-5 overflow-auto bg-surface p-3 text-sm">
            <Section title="display">
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
                    hint="two colors makes diamonds red and clubs black"
                    name="deck"
                    options={[
                        { label: "four colors", value: "four" },
                        { label: "two", value: "two" },
                    ]}
                    value={prefs.deck}
                    onChange={(deck) => set({ deck })}
                />
                <Segmented
                    legend="table"
                    name="felt"
                    options={[
                        { label: "green", value: "green" },
                        { label: "blue", value: "blue" },
                        { label: "red", value: "red" },
                        { label: "gray", value: "gray" },
                    ]}
                    value={prefs.felt}
                    onChange={(felt) => set({ felt })}
                />
                <Segmented
                    legend="card backs"
                    name="back"
                    options={[
                        { label: "red", value: "red" },
                        { label: "blue", value: "blue" },
                        { label: "black", value: "black" },
                    ]}
                    value={prefs.back}
                    onChange={(back) => set({ back })}
                />
            </Section>
            <Section title="sound">
                <Toggle
                    id="muted"
                    label="mute everything"
                    hint="the speaker at the top right does the same"
                    checked={prefs.muted}
                    onChange={(muted) => set({ muted })}
                />
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
