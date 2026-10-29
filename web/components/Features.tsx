import { ReactNode, useEffect, useRef, useState } from "react";
import classNames from "classnames";
import { Button } from "@mantine/core";
import { FiClock } from "react-icons/fi";
import Card from "./Card";
import Chips from "./Chips";
import Segmented from "./Segmented";
import StackChart from "./StackChart";
import { BotIcon } from "./Seat";
import { HandStats, LedgerRow } from "../interfaces";
import { settleLine, signed } from "../lib/ledger";
import { decks, gameTitle, games, limits } from "../lib/settings";
import { seriesColor } from "../lib/stats";

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
    felt?: boolean; // the stats chart sits on a panel, as it does at the table
    className?: string;
    children?: ReactNode;
}) {
    return (
        <li
            className={classNames(
                "flex flex-col overflow-hidden rounded-lg border border-line bg-raised transition-colors hover:border-accent/60",
                className
            )}
        >
            <div
                className={classNames(
                    "m-3 mb-0 flex min-h-40 flex-col items-center justify-center gap-3 rounded-xl p-4",
                    felt ? "felt text-on-felt" : "bg-panel"
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

function Cards({ cards, faceUp, small }: { cards: number[]; faceUp: boolean; small?: boolean }) {
    return (
        <div className="flex gap-1.5">
            {cards.map((c, i) => (
                <Card key={i} card={faceUp ? c : 0} hidden={!faceUp} placeholder={false} folded={false} small={small} order={i} />
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
    const [dealt, deal] = useDeal();
    // a new game or deck deals a new hand
    const redeal = (set: (v: string) => void) => (v: string) => {
        set(v);
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
                        <Cards cards={hands[game][deck]} faceUp={dealt} />
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
];
const botPlay: Record<string, string> = {
    easy: "a calling station: checks when it can and calls most bets.",
    medium: "tight and aggressive: raises by position, bets its draws.",
};

function Bots() {
    const [level, setLevel] = useState("medium");
    return (
        <Tile
            title="bots"
            text="fill empty seats, or practice on your own."
            stage={
                <div className="flex items-center gap-3">
                    <div className="w-32 rounded-xl bg-panel px-2 py-1.5 text-center text-fg">
                        <p className="flex items-center justify-center text-sm leading-tight">
                            harper
                            <BotIcon level={level} />
                        </p>
                        <p className="text-lg leading-tight font-semibold">1840</p>
                    </div>
                    <span className="flex items-center gap-1.5 rounded-full bg-felt-dark/80 px-2 py-0.5 text-sm font-semibold whitespace-nowrap">
                        <Chips amount={level === "easy" ? 20 : 60} bb={20} className="size-5" />
                        {level === "easy" ? "call 20" : "raise to 60"}
                    </span>
                </div>
            }
        >
            <div className="mt-2">
                <Segmented legend="level" name="demo-bot" options={botLevels} value={level} onChange={setLevel} />
            </div>
            <p className="mt-1 text-sm text-muted">{botPlay[level]}</p>
        </Tile>
    );
}

const startRows: LedgerRow[] = [
    { name: "ann", boughtIn: 2000, cashedOut: 0, stack: 2840, net: 840 },
    { name: "bob", boughtIn: 3000, cashedOut: 0, stack: 2500, net: -500 },
    { name: "cat", boughtIn: 2000, cashedOut: 1660, stack: 0, net: -340 },
    { name: "dan", boughtIn: 2000, cashedOut: 0, stack: 2000, net: 0 },
];

function Ledger() {
    const [rows, setRows] = useState(startRows);
    const [banker, setBanker] = useState("ann");
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
                                <div className="flex gap-2 text-xs text-subtle">
                                    <span>
                                        in {r.boughtIn} · out {r.cashedOut}
                                    </span>
                                    <span className="ml-auto text-muted" data-settle>
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

// sampleHands plays made-up hands between three players: each hand someone wins
// what the others put in
const players = ["ann", "bob", "cat"];
function sampleHands(count: number): HandStats[] {
    let seed = 11;
    const rand = (n: number) => {
        seed = (seed * 16807) % 2147483647;
        return seed % n;
    };
    const stacks: Record<string, number> = { ann: 2000, bob: 2000, cat: 2000 };
    const out: HandStats[] = [];
    for (let i = 0; i < count; i++) {
        const winner = players[rand(3)];
        let pot = 0;
        for (const p of players) {
            if (p === winner) continue;
            const paid = Math.min(stacks[p], 20 * (1 + rand(10)));
            stacks[p] -= paid;
            pot += paid;
        }
        stacks[winner] += pot;
        out.push({ hand: i + 1, started: i * 90_000, stacks: { ...stacks }, played: players, won: { [winner]: pot } });
    }
    return out;
}
const allHands = sampleHands(60);

function Stats() {
    const [count, setCount] = useState(20);
    return (
        <Tile
            title="stats and history"
            text="everyone's chips over the night, how they play, and every hand to look back on."
            felt={false}
            className="sm:col-span-2"
            stage={
                <div className="w-full">
                    <ul className="mb-1 flex gap-3 text-xs text-muted">
                        {players.map((n, i) => (
                            <li key={n} className="flex items-center gap-1">
                                <span className="inline-block h-0.5 w-3 rounded-full" style={{ background: seriesColor(i) }} />
                                {n}
                            </li>
                        ))}
                    </ul>
                    <StackChart hands={allHands.slice(0, count)} names={players} height={120} />
                </div>
            }
        >
            <div className="mt-2">
                <Button size="compact-xs" disabled={count >= allHands.length} onClick={() => setCount(count + 10)}>
                    play 10 more hands
                </Button>
            </div>
        </Tile>
    );
}

const rules = ["7-2 bounty", "straddle", "run it twice", "bomb pots", "double board", "time bank"] as const;
type Rule = (typeof rules)[number];
const ruleText: Record<Rule, string> = {
    "7-2 bounty": "win a hand with 7-2 and everyone pays you. tap the cards.",
    straddle: "put in two big blinds before the deal and act last before the flop.",
    "run it twice": "all in before the river? deal the rest twice, half the pot each.",
    "bomb pots": "everyone puts in, and the hand starts on the flop.",
    "double board": "a bomb pot on two boards, each wins half.",
    "time bank": "time cards for the big decisions, earned as you play.",
};
const flop = [card(king, S), card(nine, D), card(four, C)];
const boards: Record<string, number[][]> = {
    "run it twice": [
        [...flop, card(two, H), card(queen, C)],
        [...flop, card(jack, S), card(seven, H)],
    ],
    "double board": [
        [card(ace, H), card(ten, H), card(six, S), card(three, C), card(eight, D)],
        [card(queen, S), card(jack, D), card(five, H), card(two, S), card(nine, C)],
    ],
};

// pill sits on the felt like a seat's bet
const pill = "flex items-center gap-1.5 rounded-full bg-felt-dark/80 px-2 py-0.5 text-sm font-semibold";

function HouseRules() {
    const [rule, setRule] = useState<Rule>("7-2 bounty");
    const [flipped, setFlipped] = useState(false);

    let stage: ReactNode;
    if (rule === "7-2 bounty") {
        stage = (
            <>
                <button type="button" aria-label="turn over the cards" onClick={() => setFlipped(!flipped)}>
                    <Cards cards={[card(seven, C), card(two, D)]} faceUp={flipped} />
                </button>
                <span className={classNames(pill, !flipped && "invisible")}>+2 bb from everyone</span>
            </>
        );
    } else if (rule in boards) {
        stage = boards[rule].map((b, i) => <Cards key={i} cards={b} faceUp small />);
    } else if (rule === "straddle") {
        stage = (
            <span className={pill}>
                <Chips amount={40} bb={20} className="size-5" />
                straddle 40
            </span>
        );
    } else if (rule === "bomb pots") {
        stage = (
            <span className={classNames(pill, "text-base")}>
                <Chips amount={600} bb={20} className="size-6" />
                pot 600
            </span>
        );
    } else {
        stage = (
            <span className={pill}>
                <FiClock />
                +30s
            </span>
        );
    }

    return (
        <Tile
            title="house rules"
            text={ruleText[rule]}
            className="sm:col-span-2 lg:col-span-3"
            stage={<div className="flex min-h-28 flex-col items-center justify-center gap-2">{stage}</div>}
        >
            <div className="mt-2 flex flex-wrap gap-1.5" data-rules>
                {rules.map((r) => (
                    <Button
                        key={r}
                        size="compact-xs"
                        variant={r === rule ? "filled" : "default"}
                        onClick={() => {
                            setRule(r);
                            setFlipped(false);
                        }}
                    >
                        {r}
                    </Button>
                ))}
            </div>
        </Tile>
    );
}

// Features shows what the app does, under the public tables on the home page
export default function Features() {
    return (
        <section aria-label="features" className="mx-auto mt-24 w-full max-w-4xl px-4">
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
