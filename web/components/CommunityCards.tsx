import { useContext } from "react";
import classNames from "classnames";
import { AppContext } from "../providers/AppStore";
import { runWinners, winningCards } from "../lib/hands";
import { Card as CardType } from "../interfaces";
import Card from "./Card";

export default function CommunityCards() {
    const { appState } = useContext(AppContext);
    const view = appState.view;
    if (!view) return null;

    // a duck hunt fills the slots the hand didn't reach, faded: they count for nothing
    const hunted = (i: number) => view.hunted[i - view.board.length];
    // the board running out more than once deals each extra run under the first,
    // from where they split
    const runs = view.runs > 1 ? view.runs : 1;
    const winners = runWinners(view);
    // the cards before the split are on every board, so they count for any run
    const shared = new Set(view.board.slice(0, view.runFrom));
    const first = winningCards(view, runs > 1 ? 0 : undefined);
    const all = winningCards(view);
    const highlight = (c: CardType) => first.has(c) || (shared.has(c) && all.has(c));
    return (
        <div className="relative flex flex-col items-center gap-1" data-boards={runs}>
            <div className="relative flex flex-row gap-0.5 text-spade sm:gap-1" data-board>
                {[0, 1, 2, 3, 4].map((i) => (
                    <div key={i}>
                        {view.board[i] ? (
                            // show dealt card
                            <Card
                                card={view.board[i]}
                                placeholder={false}
                                folded={false}
                                hidden={false}
                                highlight={highlight(view.board[i])}
                                order={i < 3 ? i : 0}
                            />
                        ) : hunted(i) ? (
                            <div className="opacity-50" data-hunted>
                                <Card card={hunted(i)} placeholder={false} folded={false} hidden={false} />
                            </div>
                        ) : (
                            <Card card={0} placeholder={true} folded={false} hidden={false} />
                        )}
                    </div>
                ))}
                {view.hunted.length > 0 && (
                    <span className="absolute inset-x-0 top-full mt-0.5 text-center text-xs text-muted">duck hunt</span>
                )}
            </div>
            {runs > 1 && <RunCaption run={0} winner={winners[0]} />}
            {Array.from({ length: runs - 1 }, (_, k) => k + 1).map((run) => (
                <ExtraBoard
                    key={run}
                    run={run}
                    cards={(view.boards[run] ?? []).slice(view.runFrom)}
                    from={view.runFrom}
                    winning={winningCards(view, run)}
                    winner={winners[run]}
                />
            ))}
        </div>
    );
}

// ExtraBoard is one more run of the board, small, with its cards lined up
// under the slots they replace
function ExtraBoard({
    run,
    cards,
    from,
    winning,
    winner,
}: {
    run: number;
    cards: CardType[];
    from: number;
    winning: Set<CardType>;
    winner?: { names: string; hand: string };
}) {
    return (
        <div className="flex flex-col items-center" data-run={run + 1}>
            <div className="flex flex-row items-center gap-0.5 text-spade sm:gap-1">
                {[0, 1, 2, 3, 4].map((i) => (
                    <div key={i} className={classNames(i < from && "invisible")}>
                        {i >= from && cards[i - from] ? (
                            <Card
                                card={cards[i - from]}
                                placeholder={false}
                                folded={false}
                                hidden={false}
                                highlight={winning.has(cards[i - from])}
                                small
                            />
                        ) : (
                            <Card card={0} placeholder={true} folded={false} hidden={false} small />
                        )}
                    </div>
                ))}
            </div>
            <RunCaption run={run} winner={winner} />
        </div>
    );
}

// RunCaption says which run a board is, and who won it once it's settled
function RunCaption({ run, winner }: { run: number; winner?: { names: string; hand: string } }) {
    return (
        <p className="text-xs text-felt-ink/80 short:text-[0.6rem]" data-run-caption>
            run {run + 1}
            {winner && (
                <>
                    {" "}
                    · {winner.names}
                    {winner.hand && `, ${winner.hand}`}
                </>
            )}
        </p>
    );
}
