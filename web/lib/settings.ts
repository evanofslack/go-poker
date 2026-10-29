import { Settings } from "../interfaces/index";

// Presets for the settings form. The server takes any value within its
// limits (backend/server/validate.go), which these mirror.
export const games = [
    { label: "hold'em", value: "holdem" },
    { label: "Omaha", value: "omaha" },
    { label: "Omaha 5", value: "omaha5" },
];
export const limits = [
    { label: "no limit", value: "no" },
    { label: "pot limit", value: "pot" },
];
export const decks = [
    { label: "full", value: "full" },
    { label: "short (6+)", value: "short" },
];
export const stakes = ["1/2", "5/10", "10/20", "25/50", "100/200"];
export const seatCounts = [2, 6, 8, 9];
export const turnTimes = [0, 15000, 30000, 60000, 120000]; // 0 is no turn timer
export const handDelays = [2000, 3000, 5000, 8000, 10000, 15000];
export const decisionTimes = [3000, 5000, 8000, 15000, 30000]; // to answer a straddle ask or a run it vote
export const straddleModes = ["off", "optional", "forced"];
export const antes = [0, 0.1, 0.25, 0.5, 1]; // in big blinds, 0 is no ante
export const sevenDeuces = [0, 1, 2, 5]; // 7-2 bounties in big blinds, 0 is off
export const standUpPayouts = [1, 2, 5, 10]; // stand-up payouts in big blinds
export const bombPots = [1, 2, 5, 10]; // bomb pots in big blinds
export const everyHands = [5, 10, 20, 50]; // hands between scheduled bomb pots
export const everyOrbits = [1, 2, 3, 5];
export const maxBigBlind = 1_000_000;
export const maxBuyInBB = 1000;
export const maxPasswordLen = 32;

// SettingsForm is what the form edits: blinds as typed, buy-ins in big blinds
export type SettingsForm = {
    game: string; // holdem, omaha or omaha5
    limit: string; // no or pot
    deck: string; // full or short
    stakes: string; // a preset, or "custom" to use sb and bb
    sb: string;
    bb: string;
    minBuyBB: string;
    maxBuyBB: string;
    seats: number;
    turnTime: number;
    nextHandDelay: number;
    public: boolean;
    approveRebuys: boolean;
    ante: number; // in big blinds
    sevenDeuce: number; // in big blinds
    duckHunt: boolean;
    straddle: string; // off, optional or forced
    decisionTime: number;
    runIt: boolean;
    bombPot: number; // in big blinds, 0 for no bomb pots
    special: string; // which bomb pot: bomb, or double-bomb on two boards
    specialEvery: number; // 0 for only when the host calls one
    specialUnit: string; // hands or orbits
};

// the server's defaults
export const defaultForm: SettingsForm = {
    game: "holdem",
    limit: "no",
    deck: "full",
    stakes: "10/20",
    sb: "10",
    bb: "20",
    minBuyBB: "50",
    maxBuyBB: "200",
    seats: 8,
    turnTime: 30000,
    nextHandDelay: 5000,
    public: true,
    approveRebuys: false,
    ante: 0,
    sevenDeuce: 0,
    duckHunt: true,
    straddle: "off",
    decisionTime: 5000,
    runIt: true,
    bombPot: 0,
    special: "bomb",
    specialEvery: 0,
    specialUnit: "hands",
};

// blinds are the form's small and big blind
export function blinds(f: SettingsForm): [number, number] {
    const [sb, bb] = f.stakes === "custom" ? [f.sb, f.bb] : f.stakes.split("/");
    return [parseInt(sb), parseInt(bb)];
}

// settingsError says what's wrong with the form, or "" if nothing is
export function settingsError(f: SettingsForm): string {
    const [sb, bb] = blinds(f);
    const min = parseInt(f.minBuyBB);
    const max = parseInt(f.maxBuyBB);
    if (!(sb >= 1) || !(bb >= sb)) return "small blind must be between 1 and the big blind";
    if (bb > maxBigBlind) return `big blind can't be more than ${maxBigBlind}`;
    if (!(min >= 1) || !(max >= min)) return "min buy-in must be at least 1 big blind and at most the max";
    if (max > maxBuyInBB) return `max buy-in can't be more than ${maxBuyInBB} big blinds`;
    return "";
}

// anteChips is an ante of a fraction of the big blind, in chips: at least 1
// unless there's no ante
export function anteChips(ante: number, bb: number): number {
    return ante > 0 ? Math.max(1, Math.round(ante * bb)) : 0;
}

// toSettings converts a valid form to what create-table sends
export function toSettings(f: SettingsForm): Settings {
    const [sb, bb] = blinds(f);
    return {
        game: f.game,
        limit: f.limit,
        deck: f.deck,
        seats: f.seats,
        sb,
        bb,
        minBuy: parseInt(f.minBuyBB) * bb,
        maxBuy: parseInt(f.maxBuyBB) * bb,
        turnTime: f.turnTime,
        nextHandDelay: f.nextHandDelay,
        public: f.public,
        approveRebuys: f.approveRebuys,
        ante: anteChips(f.ante, bb),
        sevenDeuce: f.sevenDeuce * bb,
        duckHunt: f.duckHunt,
        straddle: f.straddle,
        decisionTime: f.decisionTime,
        runIt: f.runIt,
        bombPot: f.bombPot * bb,
        special: f.special,
        // a schedule of bomb pots needs them on
        specialEvery: f.bombPot ? f.specialEvery : 0,
        specialUnit: f.specialUnit,
    };
}

// nearest is the preset whose amount in chips comes closest to n
function nearest(presets: number[], n: number, chips: (preset: number) => number): number {
    return presets.reduce((best, p) => (Math.abs(chips(p) - n) < Math.abs(chips(best) - n) ? p : best));
}

// fromSettings is the form for a table's settings, for the host to change them
export function fromSettings(s: Settings): SettingsForm {
    const preset = `${s.sb}/${s.bb}`;
    return {
        game: s.game,
        limit: s.limit,
        deck: s.deck,
        stakes: stakes.includes(preset) ? preset : "custom",
        sb: String(s.sb),
        bb: String(s.bb),
        minBuyBB: String(Math.round(s.minBuy / s.bb)),
        maxBuyBB: String(Math.round(s.maxBuy / s.bb)),
        seats: s.seats,
        turnTime: s.turnTime,
        nextHandDelay: s.nextHandDelay,
        public: s.public,
        approveRebuys: s.approveRebuys,
        ante: nearest(antes, s.ante, (a) => anteChips(a, s.bb)),
        sevenDeuce: nearest(sevenDeuces, s.sevenDeuce, (n) => n * s.bb),
        duckHunt: s.duckHunt,
        straddle: s.straddle,
        decisionTime: s.decisionTime,
        runIt: s.runIt,
        bombPot: s.bombPot ? nearest(bombPots, s.bombPot, (n) => n * s.bb) : 0,
        special: s.special,
        specialEvery: s.specialEvery,
        specialUnit: s.specialUnit,
    };
}

// gameName names a table's game the short way, like "NLH", "PLO5" or "6+ NLH", as the server does
export function gameName(s: Pick<Settings, "game" | "limit" | "deck">): string {
    const name = { holdem: "H", omaha: "O", omaha5: "O5" }[s.game] ?? "H";
    return (s.deck === "short" ? "6+ " : "") + (s.limit === "pot" ? "PL" : "NL") + name;
}

// gameTitle spells a table's game out, like "pot limit Omaha" or "no limit 6+ hold'em"
export function gameTitle(s: Pick<Settings, "game" | "limit" | "deck">): string {
    const name = { holdem: "texas hold'em", omaha: "Omaha", omaha5: "5 card Omaha" }[s.game] ?? "texas hold'em";
    return (s.limit === "pot" ? "pot limit " : "no limit ") + (s.deck === "short" ? "6+ " : "") + name;
}

// bountyHand is the worst hand bounty's hand: 7-2, or J-6 on a short deck
export function bountyHand(s: Pick<Settings, "deck">): string {
    return s.deck === "short" ? "J-6" : "7-2";
}

// isDouble says whether a bomb pot is on two boards
export const isDouble = (kind: string) => kind === "double-bomb";

// specialName names a bomb pot, like "double board bomb pot"
export function specialName(kind: string): string {
    return { bomb: "bomb pot", "double-bomb": "double board bomb pot" }[kind] ?? "";
}
