import { everyHands, everyOrbits, isDouble } from "../lib/settings";
import Segmented from "./Segmented";
import Toggle from "./Toggle";

type schedule = { special: string; specialEvery: number; specialUnit: string };

type specialScheduleProps = schedule & {
    id: string; // tells the create form's double board switch from the host tab's
    onChange: (change: Partial<schedule>) => void;
};

// SpecialSchedule sets up bomb pots once they're on: on one board or two, and
// whether they come round on their own or only when the host calls one. In the
// create form and live in the host tab.
export default function SpecialSchedule({ id, special, specialEvery, specialUnit, onChange }: specialScheduleProps) {
    const orbits = specialUnit === "orbits";
    return (
        <>
            <Toggle
                id={id}
                label="double board"
                hint="two boards from the flop, and each pot splits between them"
                checked={isDouble(special)}
                onChange={(on) => onChange({ special: on ? "double-bomb" : "bomb" })}
            />
            <div className="flex flex-wrap items-end gap-x-2 gap-y-3">
                <Segmented
                    legend="deal them"
                    hint="when called, the host deals one from the host tab; an orbit is a hand for each player dealt in"
                    name="specialSchedule"
                    options={[
                        { label: "when called", value: "" },
                        { label: "by hands", value: "hands" },
                        { label: "by orbits", value: "orbits" },
                    ]}
                    value={specialEvery ? specialUnit : ""}
                    // a schedule starts at its shortest sensible gap
                    onChange={(u) =>
                        onChange(u ? { specialUnit: u, specialEvery: u === "orbits" ? 1 : 10 } : { specialEvery: 0 })
                    }
                />
                {specialEvery > 0 && (
                    // a new set of options starts afresh: in place, the selection
                    // lags behind them
                    <Segmented
                        key={specialUnit}
                        legend="every"
                        name="specialEvery"
                        options={(orbits ? everyOrbits : everyHands).map((n) => ({ label: String(n), value: n }))}
                        value={specialEvery}
                        onChange={(n) => onChange({ specialEvery: n })}
                    />
                )}
            </div>
        </>
    );
}
