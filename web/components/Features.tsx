import { ReactNode, useContext, useEffect, useRef, useState } from "react";
import classNames from "classnames";
import { Button, Table } from "@mantine/core";
import { FiClock } from "react-icons/fi";
import Card from "./Card";
import Chips from "./Chips";
import Segmented from "./Segmented";
import NetChart from "./NetChart";
import { BotIcon } from "./Seat";
import StyleHeader from "./StyleHeader";
import { HandStats, LedgerRow } from "../interfaces";
import { settleLine, signed } from "../lib/ledger";
import { decks, gameTitle, games, limits } from "../lib/settings";
import { seriesColor, styleColumns } from "../lib/stats";
import { AppContext } from "../providers/AppStore";

// The tiles are demos on made-up data: nothing here talks to the server. The data is
// fixed, so the prerendered page matches what the browser draws. The e2e scripts wait for
// "Open" (any case) to find a table's empty seats, so no text here says it.

// card packs a rank (deuce=0 … ace=12) and a suit the way the server sends cards
const [C, D, H, S] = [0x8000, 0x4000, 0x2000, 0x1000];
const card = (rank: number, suit: number) => (rank << 8) | suit;
const [two, three, four, five, six, seven, eight, nine, ten, jack, queen, king, ace] = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];

// Tile is one feature: a demo on its stage, then what it is
function Tile({ title, text, stage, felt = true, className, children }: {
    title: string;
    text: string;
    stage: ReactNode;
    felt?: boolean; // the ledger and chart sit on a panel, as they do at the table
    className?: string;
    children?: ReactNode;
}) {
    return (
        <li className={classNames("flex flex-col overflow-hidden rounded-lg border border-line bg-raised", className)}>
            <div
                className={classNames(
                    "m-3 mb-0 flex min-h-40 flex-col items-center justify-center gap-3",
                    felt ? "felt rounded-[2.5rem] px-6 py-5 text-on-felt" : "rounded-xl bg-panel px-3 py-4"
                )}
            >
                {stage}
            </div>
            <div className="flex grow flex-col gap-1 p-4 pt-3">
                <h3 className="font-semibold text-fg">{title}</h3>
                <p className="text-sm text-muted">{text}</p>
                {children}
            </div>
        </li>
    );
}

// useDeal turns cards face down, then over again a moment later
function useDeal() {
    const [dealt, setDealt] = useState(true);
    const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
    useEffect(() => () => clearTimeout(timer.current), []);
    const deal = () => {
        setDealt(false);
        clearTimeout(timer.current);
        timer.current = setTimeout(() => setDealt(true), 350);
    };
    return [dealt, deal] as const;
}

// Cards lie face up from the start; still ones aren't dealt in when they first show
function Cards({ cards, faceUp, small, still }: { cards: number[]; faceUp: boolean; small?: boolean; still?: boolean }) {
    return (
        <div className="flex gap-1.5">
            {cards.map((c, i) => (
                <Card
                    key={i}
                    card={faceUp ? c : 0}
                    hidden={!faceUp}
                    placeholder={false}
                    folded={false}
                    small={small}
                    order={i}
                    still={still}
                />
            ))}
        </div>
    );
}

// a hand for each game, from each deck; short deck has nothing under a six
const hands: Record<string, Record<string, number[]>> = {
    holdem: {
        full: [card(ace, S), card(five, S)],
        short: [card(nine, H), card(nine, C)],
    },
    omaha: {
        full: [card(ace, S), card(ace, D), card(four, S), card(three, D)],
        short: [card(ace, S), card(ace, D), card(seven, S), card(six, D)],
    },
    omaha5: {
        full: [card(king, H), card(queen, H), card(jack, C), card(three, D), card(two, C)],
        short: [card(king, H), card(queen, H), card(jack, C), card(ten, D), card(six, C)],
    },
};

function Games() {
    const [game, setGame] = useState("holdem");
    const [limit, setLimit] = useState("no");
    const [deck, setDeck] = useState("full");
    const [changed, setChanged] = useState(false);
    const [dealt, deal] = useDeal();
    // a new game or deck deals a new hand
    const redeal = (set: (v: string) => void) => (v: string) => {
        set(v);
        setChanged(true);
        deal();
    };

    return (
        <Tile
            title="every game"
            text="hold'em, omaha and 5 card omaha. no limit or pot limit, full deck or short."
            className="sm:col-span-2"
            stage={
                <>
                    <div key={game + deck}>
                        <Cards cards={hands[game][deck]} faceUp={dealt} still={!changed} />
                    </div>
                    <p className="text-sm">
                        {gameTitle({ game, limit, deck })}
                        <span className="opacity-70"> · {limit === "pot" ? "raise up to the pot" : "all in any time"}</span>
                    </p>
                </>
            }
        >
            <div className="mt-2 flex flex-wrap gap-x-4 gap-y-2" data-games>
                <Segmented legend="game" name="demo-game" options={games} value={game} onChange={redeal(setGame)} />
                <Segmented legend="limit" name="demo-limit" options={limits} value={limit} onChange={setLimit} />
                <Segmented legend="deck" name="demo-deck" options={decks} value={deck} onChange={redeal(setDeck)} />
            </div>
        </Tile>
    );
}

const botLevels = [
    { label: "easy", value: "easy" },
    { label: "medium", value: "medium" },
    { label: "hard", value: "hard" },
];

// each level's typical move, and how it plays
const botPlay: Record<string, { chips: number; move: string; text: string }> = {
    easy: { chips: 20, move: "call 20", text: "easy calls too much." },
    medium: { chips: 60, move: "raise to 60", text: "medium plays the odds." },
    hard: { chips: 180, move: "3-bet to 180", text: "hard reads ranges and learns how you play." },
};

// pill sits on the felt like a seat's bet
const pill = "flex items-center gap-1.5 rounded-full bg-felt-dark/80 px-2 py-0.5 text-sm font-semibold whitespace-nowrap";

function Bots() {
    const [level, setLevel] = useState("medium");
    const play = botPlay[level];
    return (
        <Tile
            title="bots"
            text={`fill empty seats, or practice on your own. ${play.text}`}
            stage={
                <div className="flex items-center gap-3">
                    <div className="w-24 rounded-xl bg-panel px-2 py-1.5 text-center text-fg">
                        <p className="flex items-center justify-center text-sm leading-tight">
                            kenny
                            <BotIcon level={level} />
                        </p>
                        <p className="text-lg leading-tight font-semibold">1840</p>
                    </div>
                    <span className={pill}>
                        <Chips amount={play.chips} bb={20} className="size-5" />
                        {play.move}
                    </span>
                </div>
            }
        >
            <div className="mt-2">
                <Segmented legend="level" name="demo-bot" options={botLevels} value={level} onChange={setLevel} />
            </div>
        </Tile>
    );
}

// The night is one made-up session the ledger and stats both show: mostly small pots,
// eric busts and buys back in, daria cashes out early.
const bb = 20;
const players = ["gus", "eric", "rishi", "daria"];
const buyIn = 2000;
const leaves: Record<string, number> = { daria: 30 }; // the hand they leave after
// big pots: hand → [winner, loser, chips the loser puts in]
const bigPots: Record<number, [string, string, number]> = {
    9: ["gus", "eric", Infinity],
    17: ["daria", "gus", 700],
    22: ["rishi", "gus", 1200],
    27: ["daria", "rishi", 500],
    34: ["eric", "rishi", 900],
};

function playNight(count: number) {
    let seed = 11;
    const rand = (n: number) => {
        seed = (seed * 16807) % 2147483647;
        return seed % n;
    };
    const stacks: Record<string, number> = {};
    const rows: Record<string, LedgerRow> = {};
    for (const p of players) {
        stacks[p] = buyIn;
        rows[p] = { name: p, boughtIn: buyIn, cashedOut: 0, stack: 0, net: 0 };
    }
    const played: HandStats[] = [];
    for (let h = 1; h <= count; h++) {
        const seated = players.filter((p) => !(leaves[p] < h));
        // a busted player buys back in
        for (const p of seated) {
            if (stacks[p] === 0) {
                stacks[p] = buyIn;
                rows[p].boughtIn += buyIn;
            }
        }
        let winner: string;
        let pot = 0;
        if (bigPots[h]) {
            const [w, loser, chips] = bigPots[h];
            winner = w;
            pot = Math.min(stacks[loser], chips);
            stacks[loser] -= pot;
        } else {
            winner = seated[rand(seated.length)];
            for (const p of seated) {
                if (p === winner) continue;
                const paid = Math.min(stacks[p], bb * rand(9));
                stacks[p] -= paid;
                pot += paid;
            }
        }
        stacks[winner] += pot;
        played.push({
            hand: h,
            started: h * 90_000,
            stacks: Object.fromEntries(seated.map((p) => [p, stacks[p]])),
            played: seated,
            won: { [winner]: pot },
            net: Object.fromEntries(seated.map((p) => [p, stacks[p] - rows[p].boughtIn])),
        });
        for (const p of seated) {
            if (leaves[p] === h) {
                rows[p].cashedOut = stacks[p];
                stacks[p] = 0;
            }
        }
    }
    const ledger = players.map((p) => {
        const r = { ...rows[p], stack: stacks[p] };
        return { ...r, net: r.stack + r.cashedOut - r.boughtIn };
    });
    return { played, ledger };
}
const night = playNight(40);

function Ledger() {
    const [rows, setRows] = useState(night.ledger);
    const [banker, setBanker] = useState("gus");
    // cashing out moves the stack off the table; the net stays as it was
    const cashOut = (name: string) =>
        setRows(rows.map((r) => (r.name === name ? { ...r, cashedOut: r.cashedOut + r.stack, stack: 0 } : r)));

    return (
        <Tile
            title="settle up"
            text="every buy-in and cash out is tracked, and everyone sees who pays whom. tap a name to make them banker."
            felt={false}
            stage={
                <div className="w-full text-sm" data-ledger>
                    <ul>
                        {rows.map((r) => (
                            <li key={r.name} className="border-b border-raised py-1 last:border-b-0">
                                <div className="flex items-center gap-2">
                                    <button
                                        type="button"
                                        className="font-semibold text-fg hover:text-accent"
                                        title={`make ${r.name} banker`}
                                        onClick={() => setBanker(r.name)}
                                    >
                                        {r.name}
                                    </button>
                                    {r.name === banker && <span className="text-xs text-accent">banker</span>}
                                    {r.stack === 0 && <span className="text-xs text-subtle">left</span>}
                                    {r.stack > 0 && (
                                        <Button size="compact-xs" variant="subtle" onClick={() => cashOut(r.name)}>
                                            cash out
                                        </Button>
                                    )}
                                    <span
                                        className={classNames(
                                            "ml-auto font-semibold",
                                            r.net > 0 && "text-gain",
                                            r.net < 0 && "text-loss"
                                        )}
                                    >
                                        {signed(r.net)}
                                    </span>
                                </div>
                                <div className="flex flex-wrap gap-x-2 text-xs text-subtle">
                                    <span className="whitespace-nowrap">
                                        in {r.boughtIn} · out {r.cashedOut}
                                    </span>
                                    <span className="ml-auto whitespace-nowrap text-muted" data-settle>
                                        {settleLine(r.name, r.net, banker)}
                                    </span>
                                </div>
                            </li>
                        ))}
                    </ul>
                </div>
            }
        />
    );
}

// how each played the night; bb/100 follows from their result
const style: Record<string, { vpip: number; pfr: number; af: string }> = {
    gus: { vpip: 24, pfr: 19, af: "3.1" },
    eric: { vpip: 52, pfr: 8, af: "0.7" },
    rishi: { vpip: 31, pfr: 22, af: "2.4" },
    daria: { vpip: 18, pfr: 12, af: "1.6" },
};
const shownStyle = ["VPIP", "PFR", "AF", "bb/100"];

function Stats() {
    const dealtIn = (p: string) => night.played.filter((h) => h.played.includes(p)).length;
    return (
        <Tile
            title="stats and history"
            text="everyone's wins and losses over the night, how they play, and every hand to look back on."
            felt={false}
            className="sm:col-span-2"
            stage={
                <div className="flex w-full flex-col gap-3">
                    <NetChart hands={night.played} names={players} height={130} />
                    <Table fz="xs" verticalSpacing={2} horizontalSpacing={4} withRowBorders={false} highlightOnHover aria-label="play style">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th className="font-normal text-subtle">player</Table.Th>
                                {styleColumns
                                    .filter(([label]) => shownStyle.includes(label))
                                    .map(([label, title]) => (
                                        <Table.Th key={label} className="text-right font-normal text-subtle">
                                            <StyleHeader label={label} title={title} />
                                        </Table.Th>
                                    ))}
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody className="text-fg tabular-nums">
                            {night.ledger.map((r, i) => (
                                <Table.Tr key={r.name}>
                                    <Table.Td>
                                        <span className="flex items-center gap-1.5 font-semibold">
                                            <span className="inline-block h-0.5 w-3 rounded-full" style={{ background: seriesColor(i) }} />
                                            {r.name}
                                        </span>
                                    </Table.Td>
                                    <Table.Td className="text-right">{style[r.name].vpip}%</Table.Td>
                                    <Table.Td className="text-right">{style[r.name].pfr}%</Table.Td>
                                    <Table.Td className="text-right">{style[r.name].af}</Table.Td>
                                    <Table.Td className="text-right">{Math.round((100 * r.net) / bb / dealtIn(r.name))}</Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </div>
            }
        />
    );
}

// Scene is one house rule on the shared felt
function Scene({ label, className, children }: { label: string; className?: string; children: ReactNode }) {
    return (
        <div className={classNames("flex flex-col items-center gap-2", className)}>
            <div className="flex min-h-16 flex-col items-center justify-center gap-1">{children}</div>
            <p className="text-xs opacity-80">{label}</p>
        </div>
    );
}

const flop = [card(king, S), card(nine, D), card(four, C)];
const runouts = [
    [...flop, card(two, H), card(queen, C)],
    [...flop, card(jack, S), card(eight, H)],
];

function HouseRules() {
    const [dealt, deal] = useDeal();
    return (
        <Tile
            title="house rules"
            text="7-2 bounties, straddles, running it twice, bomb pots on one board or two, and time cards for the big decisions."
            className="sm:col-span-2 lg:col-span-3"
            stage={
                <div className="grid w-full grid-cols-2 items-end gap-x-4 gap-y-5 sm:flex sm:flex-wrap sm:justify-around sm:gap-x-6" data-rules>
                    <Scene label="7-2 bounty">
                        <button type="button" aria-label="deal the 7-2 again" onClick={deal}>
                            <Cards cards={[card(seven, C), card(two, D)]} faceUp={dealt} small still />
                        </button>
                        <span className={classNames(pill, "text-xs")}>+2 bb from everyone</span>
                    </Scene>
                    <Scene label="straddle">
                        <span className={pill}>
                            <Chips amount={40} bb={bb} className="size-5" />
                            straddle 40
                        </span>
                    </Scene>
                    <Scene label="run it twice · double board" className="col-span-2">
                        {runouts.map((b, i) => (
                            <Cards key={i} cards={b} faceUp small still />
                        ))}
                    </Scene>
                    <Scene label="bomb pots">
                        <span className={pill}>
                            <Chips amount={600} bb={bb} className="size-6" />
                            pot 600
                        </span>
                    </Scene>
                    <Scene label="time bank">
                        <span className={pill}>
                            <FiClock />
                            +30s
                        </span>
                    </Scene>
                </div>
            }
        />
    );
}

// Features shows what the app does, under the public tables on the home page
export default function Features() {
    const { appState } = useContext(AppContext);
    // cards animate under data-motion, as at the table. Prefs live in the browser, so it's
    // set once mounted, keeping the prerendered page the same as the first render.
    const [mounted, setMounted] = useState(false);
    useEffect(() => setMounted(true), []);
    return (
        <section
            aria-label="features"
            className="mx-auto mt-24 mb-16 w-full max-w-4xl px-4"
            data-motion={mounted ? appState.prefs.motion : undefined}
        >
            <h2 className="text-2xl font-semibold text-fg">poker with friends</h2>
            <p className="mt-1 text-muted">no accounts, no downloads, just send a link.</p>
            <ul className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <Games />
                <Bots />
                <Ledger />
                <Stats />
                <HouseRules />
            </ul>
        </section>
    );
}
