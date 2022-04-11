import classNames from "classnames";
import { Card as CardType } from "../interfaces";

type cardProps = {
    card: CardType;
    placeholder: boolean;
    folded: boolean;
};

function cardToString(card: CardType) {
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

    // opponents cards represented with "?"
    if (card === "?") {
        return "?";
    }

    let c = parseInt(card);
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

function color(card: CardType) {
    return classNames(
        {
            "text-red-700": card[1] == "H",
            "text-blue-700": card[1] == "D",
            "text-green-700": card[1] == "C",
            "text-black": card[1] == "S",
        },
        "rounded-md border border-zinc-100 shadow-2xl bg-white pt-1 px-2.5 text-5xl font-normal w-16 h-24 flex items-center justify-start flex-col"
    );
}

export default function Card({ card, placeholder, folded }: cardProps) {
    if (placeholder) {
        return (
            <div className="flex h-24 w-16 items-center justify-center rounded-md bg-green-900 opacity-20"></div>
        );
    }

    const c = cardToString(card);
    if (c == "2\u0000" || card == "0") {
        return null;
    }
    if (c === "?") {
        if (folded) {
            // return null;
            return <div className={"flex h-24 w-16  "}></div>;
        }
        return (
            <div
                className={
                    "flex h-24 w-16 items-center justify-center rounded-md border-4 border border-white bg-red-900"
                }
            ></div>
        );
    }
    if (folded) {
        return (
            <div className={color(c)}>
                <div
                    className={
                        "flex w-full items-start justify-start text-3xl font-semibold opacity-40"
                    }
                >
                    {c[0]}
                </div>
                <div className="opacity-40">{getSuitChar(c[1])}</div>
            </div>
        );
    }
    return (
        <div className={color(c)}>
            <div className={"flex w-full items-start justify-start text-3xl font-semibold"}>
                {c[0]}
            </div>
            <div>{getSuitChar(c[1])}</div>
        </div>
    );
}
