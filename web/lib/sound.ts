import { GameEvent } from "../interfaces";
import { Prefs } from "./prefs";

// Sounds are Kenney's (public/sounds/LICENSE.txt), in groups that the options
// tab turns on and off. Each play also fires a poker-sound event on window,
// with the group and file, so the e2e tests can hear them.
export type Group = "turn" | "timer" | "decisions" | "cards" | "chips" | "chat";

// samples are what each group's test button plays
export const samples: Record<Group, string> = {
    turn: "glass-002",
    timer: "tick-002",
    decisions: "question-001",
    cards: "card-slide-1",
    chips: "chips-stack-1",
    chat: "pluck-001",
};

const slides = ["card-slide-1", "card-slide-2", "card-slide-3"];
const places = ["card-place-1", "card-place-2", "card-place-3"];
const stacks = ["chips-stack-1", "chips-stack-2", "chips-stack-3", "chips-stack-4"];
const all = [...slides, ...places, ...stacks, "card-place-4", "card-shove-1", "chips-handle-5", "click-002", "chips-collide-1", ...Object.values(samples)];

// your own sounds keep playing in a hidden tab; the table's and chat go quiet
const yours: Group[] = ["turn", "timer", "decisions"];

let prefs: Prefs | null = null;
let ctx: AudioContext | null = null;
const buffers = new Map<string, Promise<AudioBuffer | null>>();
const last: Partial<Record<Group, number>> = {};

// setSoundPrefs is how the app tells sound what the viewer chose
export function setSoundPrefs(p: Prefs) {
    prefs = p;
}

// audio waits for the viewer's first gesture, so resume on each one
function context() {
    if (!ctx) {
        try {
            ctx = new AudioContext();
            const resume = () => ctx?.resume();
            window.addEventListener("pointerdown", resume);
            window.addEventListener("keydown", resume);
        } catch {
            return null;
        }
    }
    return ctx;
}

function load(name: string) {
    let b = buffers.get(name);
    if (!b) {
        const c = context();
        b = c
            ? fetch(`/sounds/${name}.mp3`)
                  .then((r) => r.arrayBuffer())
                  .then((data) => c.decodeAudioData(data))
                  .catch(() => null)
            : Promise.resolve(null);
        buffers.set(name, b);
    }
    return b;
}

// preload fetches and decodes every sound, so the first of each isn't late
export function preload() {
    all.forEach(load);
}

function on(group: Group, p: Prefs) {
    if (group === "turn") return p.turn === "always" || (p.turn === "away" && document.hidden);
    return p[group];
}

// play plays one of names, picked at random, after delay ms. It does nothing
// when muted, the group is off, or the tab is hidden and it isn't yours, and
// skips a sound within 60ms of the last one in its group unless it's delayed.
// force plays it anyway, for the test buttons.
export function play(group: Group, names: string | string[], { delay = 0, force = false } = {}) {
    const p = prefs;
    if (!p) return;
    if (!force) {
        if (p.muted || !on(group, p) || (document.hidden && !yours.includes(group))) return;
        const now = performance.now();
        if (!delay && now - (last[group] ?? -Infinity) < 60) return;
        if (!delay) last[group] = now;
    }
    const list = typeof names === "string" ? [names] : names;
    const name = list[Math.floor(Math.random() * list.length)];
    window.dispatchEvent(new CustomEvent("poker-sound", { detail: { group, name } }));

    const c = context();
    if (!c) return;
    load(name).then((buffer) => {
        if (!buffer) return;
        const source = c.createBufferSource();
        const gain = c.createGain();
        source.buffer = buffer;
        gain.gain.value = p.volume;
        source.connect(gain).connect(c.destination);
        source.start(c.currentTime + delay / 1000);
    });
}

// playEvent plays what a table event sounds like
export function playEvent(ev: GameEvent) {
    switch (ev.type) {
        case "hand-started":
            // two cards each, like the deal animation
            play("cards", slides);
            play("cards", slides, { delay: 80 });
            return;
        case "street-dealt":
            ev.data.cards.forEach((_, i) => play("cards", places, { delay: i * 120 }));
            return;
        case "acted": {
            const { kind, allIn } = ev.data;
            if (kind === "fold") play("cards", "card-shove-1");
            else if (kind === "check") play("chips", "click-002");
            else if (allIn) play("chips", "chips-handle-5");
            else play("chips", stacks);
            return;
        }
        case "shown":
            play("cards", "card-place-4");
            return;
        case "pot-won":
            play("chips", "chips-collide-1");
            return;
    }
}
