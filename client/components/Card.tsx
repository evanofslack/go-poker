import Image from "next/image";
import { getCardSVG } from "../util/cardDrawer";

type cardProps = {
    card: string;
    width: number;
    height: number;
};

export default function Card({ card, width, height }: cardProps) {
    if (card == "2\u0000") {
        return null;
    }

    return <Image src={getCardSVG(card)} width={width} height={height} />;
}
