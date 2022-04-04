export function getCardSVG(card: string) {
    if (card === "?") {
        return `/images/cards/1B.svg`; // back of card
    }
    return `/images/cards/${card.toUpperCase()}.svg`;
}

export function drawCards(cards: string[]) {
    // return cards.filter((card) => card !== "2\u0000").map((card) => getCardSVG(card));
    return cards.filter((card) => card != "2\u0000");
}
