import classNames from "classnames";
import { Card as CardType } from "../interfaces";

type cardProps = {
    card: CardType;
    placeholder: boolean;
    folded: boolean;
    hidden: boolean;
    highlight?: boolean;
    big?: boolean; // your own cards, a size up from a tablet on
};

function cardToString(c: CardType) {
    // convert int32 cards to a unicode string that we can display directly

    // card is the type representing a single laying card. It is 32 bits long, packed according to the following schematic:
    // 	+--------+--------+--------+--------+
    // 	|xxxbbbbb|bbbbbbbb|cdhsrrrr|xxpppppp|
    // 	+--------+--------+--------+--------+
    //
    // 	p 	= prime number of rank (deuce=2,trey=3,four=5,...,ace=41)
    // 	r 	= rank of card (deuce=0,trey=1,four=2,five=3,...,ace=12)
    // 	cdhs	= suit of card (bit turned on based on suit of card)
    // 	b 	= bit turned on depending on rank of card

    // bitwise operations to map int32 representation to string representation
    let rank = (c >> 8) & 0x0f;
    let suit = c & 0xf000;

    const numToCharRanks = ["2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"];
    const numToCharSuits = new Map();
    numToCharSuits.set(0x8000, "C");
    numToCharSuits.set(0x4000, "D");
    numToCharSuits.set(0x2000, "H");
    numToCharSuits.set(0x1000, "S");

    return numToCharRanks[rank] + numToCharSuits.get(suit);
}

const rankNames: Record<string, string> = {
    T: "ten",
    J: "jack",
    Q: "queen",
    K: "king",
    A: "ace",
};
const suitNames: Record<string, string> = { H: "hearts", D: "diamonds", C: "clubs", S: "spades" };

// cardName reads a card string like "TH" out loud, as "ten of hearts"
function cardName(c: string) {
    return `${rankNames[c[0]] ?? c[0]} of ${suitNames[c[1]]}`;
}

function getSuitChar(letter: string) {
    // convert letter suit to unicode symbol
    switch (letter) {
        case "H":
            return "\u2665";
        case "D":
            return "\u2666";
        case "C":
            return "\u2663";
        case "S":
            return "\u2660";
    }
}

// cardSize scales cards down on small screens. A phone held upright has room for a size
// up, since other players' face-down cards shrink onto their seats there.
export const cardSize = {
    h: "h-14 sm:h-[4.5rem] lg:h-24 phone:h-[4.5rem] short:h-10",
    box: "w-10 h-14 sm:w-12 sm:h-[4.5rem] lg:w-16 lg:h-24 phone:w-12 phone:h-[4.5rem] short:w-7 short:h-10",
};
export const bigCardSize = {
    h: "h-14 sm:h-24 lg:h-28 phone:h-[4.5rem] short:h-10",
    box: "w-10 h-14 sm:w-16 sm:h-24 lg:w-20 lg:h-28 phone:w-12 phone:h-[4.5rem] short:w-7 short:h-10",
};
const rankSize = "text-lg sm:text-xl lg:text-3xl phone:text-xl short:text-sm";
const bigRankSize = "text-lg sm:text-3xl lg:text-4xl phone:text-xl short:text-sm";

function color(card: string, highlight = false, big = false) {
    return classNames(
        {
            "ring-2 ring-amber-300 lg:ring-4": highlight,
            "text-red-700": card[1] == "H",
            "text-blue-700": card[1] == "D",
            "text-green-700": card[1] == "C",
            "text-black": card[1] == "S",
        },
        "rounded-md border border-zinc-100 shadow-2xl bg-white pt-0.5 px-1 text-2xl sm:text-3xl lg:pt-1 lg:px-2.5 lg:text-5xl phone:text-3xl short:text-base short:px-0.5 short:pt-0 font-normal flex items-center justify-start flex-col",
        big ? "sm:text-5xl lg:text-6xl phone:text-3xl short:text-base" : "",
        big ? bigCardSize.box : cardSize.box
    );
}

// MiniCards stand in for another player's face-down cards, tucked into the
// corner of their seat
export function MiniCards({ count }: { count: number }) {
    return (
        <div
            role="img"
            aria-label="face-down cards"
            className="absolute -top-2 right-3 z-20 flex gap-0.5"
        >
            {Array.from({ length: count }, (_, i) => (
                <div key={i} className="h-5 w-3.5 rounded-sm border border-white bg-red-900 lg:h-6 lg:w-4" />
            ))}
        </div>
    );
}

export default function Card({ card, placeholder, folded, hidden, highlight, big = false }: cardProps) {
    const box = big ? bigCardSize.box : cardSize.box;
    const rank = big ? bigRankSize : rankSize;
    if (placeholder) {
        return (
            <div
                aria-hidden
                className={classNames(cardSize.box, "rounded-md bg-green-900 opacity-20")}
            ></div>
        );
    }

    const c = cardToString(card);
    // the server sends 0 for cards this client can't see, which still show face down
    const faceDown = hidden && !folded;
    if (!faceDown && !card) {
        return null;
    }
    if (hidden) {
        if (folded) {
            return <div aria-hidden className={box}></div>;
        }
        return (
            <div
                role="img"
                aria-label="face-down card"
                className={classNames(
                    box,
                    "rounded-md border-2 border-white bg-red-900 lg:border-4"
                )}
            ></div>
        );
    }
    if (folded) {
        return (
            <div role="img" aria-label={cardName(c) + ", folded"} className={color(c, false, big)}>
                <div
                    className={classNames(
                        "flex w-full items-start justify-start font-semibold opacity-40",
                        rank
                    )}
                >
                    {c[0]}
                </div>
                <div className="opacity-40">{getSuitChar(c[1])}</div>
            </div>
        );
    }
    return (
        <div
            role="img"
            aria-label={cardName(c) + (highlight ? ", in the winning hand" : "")}
            className={color(c, highlight, big)}
        >
            <div
                className={classNames("flex w-full items-start justify-start font-semibold", rank)}
            >
                {c[0]}
            </div>
            <div>{getSuitChar(c[1])}</div>
        </div>
    );
}
