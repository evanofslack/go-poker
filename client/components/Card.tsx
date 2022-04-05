import Image from "next/image";
import { getCardSVG } from "../util/cardDrawer";
import classNames from "classnames";

type cardProps = {
    card: string;
    width: number;
    height: number;
};

function getSuitChar(letter: string) {
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

function color(card: string) {
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
    if (card == "2\u0000") {
        return null;
    }
    // if (card === "?") return <div className={`bg-white w-${width} h-${height}`}>?</div>;
    if (card === "?")
        return (
            <div className="flex h-24 w-16 items-center justify-center rounded-md border-4 border border-white bg-red-900"></div>
        );

    return (
        <div className={color(card)}>
            <div className="flex w-full items-start justify-start text-3xl font-semibold">
                {card[0]}
            </div>
            <div>{getSuitChar(card[1])}</div>
        </div>
    );

    return <Image src={getCardSVG(card)} width={width} height={height} />;
}
