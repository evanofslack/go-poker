import { Settings } from "../interfaces/index";

// Presets for the create table form. The server takes any value within its
// limits (backend/server/validate.go), which these mirror.
export const stakes = ["1/2", "5/10", "10/20", "25/50", "100/200"];
export const seatCounts = [2, 6, 8, 9];
export const turnTimes = [0, 15000, 30000, 60000, 120000]; // 0 is no turn timer
export const handDelays = [2000, 3000, 5000, 8000, 10000, 15000];
export const maxBigBlind = 1_000_000;
export const maxBuyInBB = 1000;
export const maxPasswordLen = 32;

// SettingsForm is what the form edits: blinds as typed, buy-ins in big blinds
export type SettingsForm = {
    stakes: string; // a preset, or "custom" to use sb and bb
    sb: string;
    bb: string;
    minBuyBB: string;
    maxBuyBB: string;
    seats: number;
    turnTime: number;
    nextHandDelay: number;
    public: boolean;
};

// the server's defaults
export const defaultForm: SettingsForm = {
    stakes: "10/20",
    sb: "10",
    bb: "20",
    minBuyBB: "50",
    maxBuyBB: "200",
    seats: 8,
    turnTime: 30000,
    nextHandDelay: 5000,
    public: false,
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

// toSettings converts a valid form to what create-table sends
export function toSettings(f: SettingsForm): Settings {
    const [sb, bb] = blinds(f);
    return {
        seats: f.seats,
        sb,
        bb,
        minBuy: parseInt(f.minBuyBB) * bb,
        maxBuy: parseInt(f.maxBuyBB) * bb,
        turnTime: f.turnTime,
        nextHandDelay: f.nextHandDelay,
        public: f.public,
    };
}
