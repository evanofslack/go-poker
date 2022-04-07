import Image from "next/image";
import { getCardSVG } from "../util/cardDrawer";
import classNames from "classnames";
import { Card as CardType } from "../interfaces";

type cardProps = {
    card: CardType;
    width: number;
    height: number;
};

function cardToString(card: CardType) {
    // convert int representation of card from backend to character representation

    // opponents cards represented with "?"
    if (card === "?") {
        return "?";
    }
    let c = parseInt(card);
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

export default function Card({ card, width, height }: cardProps) {
    const c = cardToString(card);
    if (c == "2\u0000" || card == "0") {
        return null;
    }
    if (c === "?")
        return (
            <div className="flex h-24 w-16 items-center justify-center rounded-md border-4 border border-white bg-red-900"></div>
        );

    return (
        <div className={color(c)}>
            <div className="flex w-full items-start justify-start text-3xl font-semibold">
                {c[0]}
                {/* {cardToString(card)} */}
            </div>
            <div>{getSuitChar(c[1])}</div>
        </div>
    );

    return <Image src={getCardSVG(card)} width={width} height={height} />;
}
