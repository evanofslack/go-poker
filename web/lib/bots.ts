// botNames are first names of characters from HBO's Industry
const botNames = [
    "harper",
    "eric",
    "yasmin",
    "robert",
    "gus",
    "rishi",
    "venetia",
    "kenny",
    "daria",
    "sweetpea",
    "henry",
    "jesse",
];

// randomBotName picks a bot name nobody at the table has, ignoring case
export function randomBotName(taken: string[]) {
    const used = new Set(taken.map((n) => n.toLowerCase()));
    const free = botNames.filter((n) => !used.has(n));
    const names = free.length > 0 ? free : botNames;
    return names[Math.floor(Math.random() * names.length)];
}
