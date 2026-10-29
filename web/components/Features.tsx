import { ReactNode } from "react";
import { FiLink } from "react-icons/fi";
import Card from "./Card";
import Chips from "./Chips";
import { signed } from "../lib/ledger";
import { seriesColor } from "../lib/stats";

// card packs a rank (deuce=0 … ace=12) and a suit bit the way the server sends cards
const card = (rank: number, suit: number) => (rank << 8) | suit;
const [clubs, diamonds] = [0x8000, 0x4000];

// a few players' chips over some hands, for the stats tile
const lines = ["0,28 20,24 40,26 60,16 80,18 100,8 120,10", "0,20 20,22 40,14 60,20 80,26 100,24 120,30", "0,24 20,28 40,30 60,26 80,22 100,26 120,22"];

const nets: [string, number][] = [
    ["ann", 840],
    ["bob", -500],
    ["cat", -340],
];

// pill looks like a compact button, but the tiles only show things off
const pill = "rounded-sm border border-line px-2 py-0.5 text-xs";

const features: { title: string; text: string; visual: ReactNode }[] = [
    {
        title: "invite by link",
        text: "share a link, friends pick a name and sit. password optional.",
        visual: (
            <span className="flex items-center gap-2 rounded-md bg-field px-3 py-1.5 text-sm text-muted">
                <FiLink className="shrink-0" />
                …/?table=friday
            </span>
        ),
    },
    {
        title: "host controls",
        text: "approve buy-ins, adjust chips, pause, end the session.",
        visual: (
            <span className="flex items-center gap-2 text-sm">
                <span className="whitespace-nowrap text-fg">bob · 2,000</span>
                <span className={`${pill} bg-accent text-on-accent border-accent`}>approve</span>
                <span className={`${pill} text-muted`}>deny</span>
            </span>
        ),
    },
    {
        title: "settle up",
        text: "a ledger tracks every buy-in, so nobody does math at the end.",
        visual: (
            <span className="grid w-32 grid-cols-2 text-sm">
                {nets.map(([name, net]) => (
                    <span key={name} className="contents">
                        <span className="text-fg">{name}</span>
                        <span className={`text-right ${net > 0 ? "text-gain" : "text-loss"}`}>{signed(net)}</span>
                    </span>
                ))}
            </span>
        ),
    },
    {
        title: "stats and history",
        text: "chip graphs, vpip/pfr, and every hand to replay.",
        visual: (
            <svg viewBox="0 0 120 36" className="h-12 w-36" fill="none" strokeWidth={2} strokeLinejoin="round">
                {lines.map((points, i) => (
                    <polyline key={i} points={points} stroke={seriesColor(i)} />
                ))}
            </svg>
        ),
    },
    {
        title: "bots",
        text: "fill empty seats or practice, easy or medium.",
        visual: (
            <span className="flex items-center gap-3">
                <Chips amount={300} bb={20} className="size-6" />
                <span className="rounded-full bg-badge px-2 py-0.5 text-xs text-ink">bot · medium</span>
            </span>
        ),
    },
    {
        title: "house rules",
        text: "straddles, run it twice, and the 7-2 game.",
        visual: (
            <span className="flex gap-1">
                <Card card={card(5, clubs)} placeholder={false} folded={false} hidden={false} small />
                <Card card={card(0, diamonds)} placeholder={false} folded={false} hidden={false} small />
            </span>
        ),
    },
];

// Features shows what the app does, under the join form on the home page
export default function Features() {
    return (
        <section aria-label="features" className="mx-auto w-full max-w-3xl px-4 pt-10">
            <h2 className="text-xl font-semibold text-fg">poker with friends</h2>
            <p className="mt-1 text-muted">no accounts, no downloads, just send a link.</p>
            <ul className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {features.map((f) => (
                    <li key={f.title} className="rounded-md bg-raised p-4">
                        <div aria-hidden className="flex h-20 items-center">
                            {f.visual}
                        </div>
                        <h3 className="mt-2 font-semibold text-fg">{f.title}</h3>
                        <p className="mt-1 text-sm text-muted">{f.text}</p>
                    </li>
                ))}
            </ul>
        </section>
    );
}
