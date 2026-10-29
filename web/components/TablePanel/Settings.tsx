import { ReactNode, useContext, useState } from "react";
import { FiCopy } from "react-icons/fi";
import { AppContext } from "../../providers/AppStore";
import { copyInvite } from "../../lib/invite";
import { bountyHand, gameTitle, isDouble } from "../../lib/settings";
import { Button } from "@mantine/core";
import EditSettings from "../EditSettings";
import Section from "../Section";

const chips = (n: number) => n.toLocaleString("en");
const seconds = (ms: number) => `${ms / 1000}s`;
const onOff = (on: boolean) => (on ? "on" : "off");

type row = [string, ReactNode];

// Rows lists a section's settings, a label and its value on each line
function Rows({ rows }: { rows: row[] }) {
    return (
        <dl>
            {rows.map(([label, value]) => (
                <div key={label} className="flex gap-2 border-b border-raised py-1.5 last:border-b-0">
                    <dt className="text-subtle">{label}</dt>
                    <dd className="ml-auto min-w-0 text-right text-fg tabular-nums">{value}</dd>
                </div>
            ))}
        </dl>
    );
}

// Settings shows how the host set up the table, in the sections and words of the
// form the host sets it with, and lets the host change it
export default function Settings() {
    const { appState } = useContext(AppContext);
    const [editing, setEditing] = useState(false);
    const game = appState.view;
    const table = appState.table;
    if (!game || !table) return null;
    const s = game.settings;
    const host = game.host === appState.username && !game.ended;
    const bb = (n: number) => Math.round(n / s.bb);

    const pace: row[] = [["turn timer", s.turnTime ? seconds(s.turnTime) : "off"]];
    if (s.turnTime && s.timeMax) {
        pace.push(
            ["time bank", s.timeBank === 1 ? "1 card" : `${s.timeBank} cards`],
            ["time card", seconds(s.timeCard)],
            ["earn a card", s.timeEvery ? `every ${s.timeEvery} hands` : "never"]
        );
    }
    pace.push(["between hands", seconds(s.nextHandDelay)]);

    const rules: row[] = [];
    if (s.game === "holdem") rules.push([`${bountyHand(s)} bounty`, s.sevenDeuce ? `${chips(s.sevenDeuce)} each` : "off"]);
    rules.push(["straddles", s.straddle], ["run it", s.runIt ? "up to 4 times" : "off"]);
    if (s.straddle !== "off" || s.runIt) rules.push(["to decide", seconds(s.decisionTime)]);
    rules.push(["duck hunt", s.duckHunt ? "allowed" : "off"]);

    const bombs: row[] = s.bombPot
        ? [
              ["bomb pot", `${chips(s.bombPot)} (${bb(s.bombPot)} bb)`],
              ["double board", onOff(isDouble(s.special))],
              ["deal them", s.specialEvery ? `every ${s.specialEvery} ${s.specialUnit}` : "when called"],
          ]
        : [["bomb pots", "off"]];

    return (
        <div aria-label="table settings" className="flex h-full flex-col gap-6 overflow-auto p-3 text-sm">
            <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                    <span className="truncate font-semibold text-fg">{table}</span>
                    <Button
                        variant="subtle"
                        size="compact-sm"
                        className="shrink-0"
                        leftSection={<FiCopy />}
                        title="copy invite link"
                        aria-label="copy invite link"
                        onClick={() => copyInvite(table, appState.password)}
                    >
                        invite
                    </Button>
                    {host && (
                        <Button size="xs" className="ml-auto shrink-0" onClick={() => setEditing(true)}>
                            edit
                        </Button>
                    )}
                </div>
                <p className="text-xs text-subtle">
                    hosted by <span className="text-muted">{game.host}</span>
                </p>
                {game.next && <p className="text-xs text-subtle">changes apply after this hand</p>}
            </div>
            {editing && <EditSettings onClose={() => setEditing(false)} />}
            <Section title="game">
                <Rows
                    rows={[
                        ["game", gameTitle(s)],
                        ["stakes", `${chips(s.sb)}/${chips(s.bb)}`],
                        [
                            "buy-in",
                            <>
                                {chips(s.minBuy)}–{chips(s.maxBuy)}
                                <span className="text-subtle">
                                    {" "}
                                    ({bb(s.minBuy)}–{bb(s.maxBuy)} bb)
                                </span>
                            </>,
                        ],
                        ["ante", s.ante ? `${Math.round((s.ante / s.bb) * 100)}% · ${chips(s.ante)}` : "off"],
                        ["seats", s.seats],
                    ]}
                />
            </Section>
            <Section title="access">
                <Rows
                    rows={[
                        ["public", s.public ? "yes" : "no"],
                        ["password", onOff(game.hasPassword)],
                        ["approve rebuys", onOff(s.approveRebuys)],
                    ]}
                />
            </Section>
            <Section title="pace">
                <Rows rows={pace} />
            </Section>
            <Section title="house rules">
                <Rows rows={rules} />
            </Section>
            <Section title="bomb pots">
                <Rows rows={bombs} />
            </Section>
        </div>
    );
}
