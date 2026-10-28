import { everyHands, everyOrbits, isBomb, specialKinds } from "../lib/settings";
import Segmented from "./Segmented";

type schedule = { special: string; specialEvery: number; specialUnit: string };

type specialScheduleProps = schedule & {
    bombs: boolean; // bomb pots are on, so the schedule can deal them
    onChange: (change: Partial<schedule>) => void;
};

// kindsFor are the special hands a table can deal: double board always, bomb
// pots only when they're on
export const kindsFor = (bombs: boolean) => specialKinds.filter((k) => bombs || !isBomb(k.value));

// SpecialSchedule picks how often special hands come round on their own, and
// which kind: in the create form and live in the host tab
export default function SpecialSchedule({ special, specialEvery, specialUnit, bombs, onChange }: specialScheduleProps) {
    const orbits = specialUnit === "orbits";
    const kinds = kindsFor(bombs);
    return (
        <>
            <div className="flex flex-wrap items-end gap-x-2 gap-y-3">
                <Segmented
                    legend="every"
                    hint="an orbit is a hand for each player dealt in"
                    name="specialEvery"
                    options={(orbits ? everyOrbits : everyHands).map((n) => ({ label: n ? String(n) : "off", value: n }))}
                    value={specialEvery}
                    onChange={(n) => onChange({ specialEvery: n })}
                />
                <Segmented
                    legend="counted in"
                    name="specialUnit"
                    options={[
                        { label: "hands", value: "hands" },
                        { label: "orbits", value: "orbits" },
                    ]}
                    value={specialUnit}
                    // keep a schedule going in the new unit, at its shortest sensible gap
                    onChange={(u) =>
                        onChange({ specialUnit: u, specialEvery: specialEvery ? (u === "orbits" ? 1 : 10) : 0 })
                    }
                />
            </div>
            {specialEvery > 0 && (
                <Segmented
                    legend="deal"
                    name="special"
                    options={kinds}
                    value={kinds.some((k) => k.value === special) ? special : kinds[0].value}
                    onChange={(k) => onChange({ special: k })}
                />
            )}
        </>
    );
}
