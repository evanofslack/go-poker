package server

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/alexclewontin/riverboat/eval"
	"github.com/evanofslack/go-poker/poker"
)

// handsPerPage is how many hands of history go out at once, on joining and
// for each get-hands
const handsPerPage = 20

// handRecord is one hand as it was played. Records keep every player's hole
// cards, so they only leave the table through viewFor.
type handRecord struct {
	Hand    int          `json:"hand"`    // counts from 1, like handStats
	Started int64        `json:"started"` // unix ms
	SB      uint         `json:"sb"`
	BB      uint         `json:"bb"`
	Dealer  string       `json:"dealer"`
	Players []handPlayer `json:"players"` // dealt in, in seat order
	Actions []handAction `json:"actions"` // blinds first
	Board   []string     `json:"board"`
	Pots    []handPot    `json:"pots"`
}

type handPlayer struct {
	Name  string   `json:"name"`
	UUID  string   `json:"-"`
	Seat  uint     `json:"seat"`
	Stack uint     `json:"stack"`           // before the blinds
	Cards []string `json:"cards,omitempty"` // hidden from others unless Shown
	Shown bool     `json:"shown"`           // at showdown
	Hand  string   `json:"hand,omitempty"`  // their best hand, if Shown
	Net   int      `json:"net"`             // chips won or lost
	score int      // of Hand, lower is better
}

type handAction struct {
	Street  string `json:"street"` // preflop, flop, turn or river
	Name    string `json:"name"`
	Kind    string `json:"kind"`             // small blind, big blind, check, call, bet, raise or fold
	Amount  uint   `json:"amount,omitempty"` // chips put in
	To      uint   `json:"to,omitempty"`     // their bet on this street after putting them in
	AllIn   bool   `json:"allIn,omitempty"`
	Timeout bool   `json:"timeout,omitempty"` // the turn timer acted for them
}

type handPot struct {
	Amount  uint     `json:"amount"`
	Winners []string `json:"winners"`
	Hand    string   `json:"hand,omitempty"` // empty when won without a showdown
}

var streets = map[poker.GameStage]string{
	poker.PreFlop: "preflop",
	poker.Flop:    "flop",
	poker.Turn:    "turn",
	poker.River:   "river",
}

// cardNames writes cards like "Ks", leaving out cards not dealt yet
func cardNames(cards []eval.Card) []string {
	var names []string
	for _, c := range cards {
		if c != 0 {
			names = append(names, c.String())
		}
	}
	return names
}

// viewFor is r as the player with uuid may see it: their own cards and any
// shown at showdown. Spectators pass "".
func (r handRecord) viewFor(uuid string) handRecord {
	r.Players = append([]handPlayer{}, r.Players...)
	for i, p := range r.Players {
		if !p.Shown && (uuid == "" || p.UUID != uuid) {
			r.Players[i].Cards = nil
			r.Players[i].Hand = ""
		}
	}
	return r
}

// played names everyone dealt into r
func (r *handRecord) played() []string {
	var names []string
	if r != nil {
		for _, p := range r.Players {
			names = append(names, p.Name)
		}
	}
	return names
}

// aggressor names who made the last bet or raise before street, "" if nobody
// did. Actions are in order, so the first on street ends the search.
func (r *handRecord) aggressor(street string) string {
	name := ""
	if r != nil {
		for _, a := range r.Actions {
			if a.Street == street {
				break
			}
			if a.Kind == "bet" || a.Kind == "raise" {
				name = a.Name
			}
		}
	}
	return name
}

// startRecord starts recording the hand just dealt, with its blinds. Caller
// holds t.mu.
func startRecord(t *table, view *poker.GameView) {
	r := &handRecord{
		Hand:    t.handCount + 1,
		Started: time.Now().UnixMilli(),
		SB:      view.Config.SmallBlind,
		BB:      view.Config.BigBlind,
		Dealer:  view.Players[view.DealerNum].Username,
	}
	for _, p := range view.Players {
		if p.In {
			r.Players = append(r.Players, handPlayer{
				Name:  p.Username,
				UUID:  p.UUID,
				Seat:  p.SeatID,
				Stack: p.Stack + p.TotalBet,
				Cards: cardNames(p.Cards[:]),
			})
		}
	}
	for _, b := range []struct {
		pn   uint
		kind string
	}{{view.SBNum, "small blind"}, {view.BBNum, "big blind"}} {
		p := view.Players[b.pn]
		r.Actions = append(r.Actions, handAction{
			Street: "preflop",
			Name:   p.Username,
			Kind:   b.kind,
			Amount: p.Bet,
			To:     p.Bet,
			AllIn:  p.Stack == 0,
		})
	}
	t.current = r
}

// recordAction adds pn's action to the hand. view is from before the action,
// for its street and pn's stack and bet then. Caller holds t.mu.
func recordAction(t *table, view *poker.GameView, pn uint, kind string, amount uint, timeout bool) {
	if t.current == nil {
		return
	}
	p := view.Players[pn]
	a := handAction{
		Street:  streets[view.Stage],
		Name:    p.Username,
		Kind:    kind,
		Amount:  amount,
		AllIn:   amount > 0 && amount == p.Stack,
		Timeout: timeout,
	}
	if amount > 0 {
		a.To = p.Bet + amount
	}
	t.current.Actions = append(t.current.Actions, a)
}

// finishRecord completes the hand that just ended, adds it to the totals,
// keeps it and sends it to everyone as they may see it. It returns the hand,
// nil if none was being recorded. Caller holds t.mu.
func finishRecord(t *table, view *poker.GameView) *handRecord {
	r := t.current
	if r == nil {
		return nil
	}
	t.current = nil

	r.Board = cardNames(view.CommunityCards)
	for _, pot := range view.Pots {
		if pot.Amt == 0 || len(pot.WinningPlayerNums) == 0 {
			continue
		}
		hp := handPot{Amount: pot.Amt, Hand: handName(pot.WinningScore, pot.WinningHand)}
		for _, pn := range pot.WinningPlayerNums {
			hp.Winners = append(hp.Winners, view.Players[pn].Username)
		}
		r.Pots = append(r.Pots, hp)
	}

	shown := map[string]bool{}
	for _, pn := range t.game.ShownPlayerNums() {
		shown[view.Players[pn].UUID] = true
	}
	for i, hp := range r.Players {
		p := &r.Players[i]
		end := 0
		for _, q := range view.Players {
			if q.UUID == hp.UUID {
				end = int(q.Stack)
			}
		}
		p.Net = end - int(hp.Stack)
		if shown[hp.UUID] && len(view.CommunityCards) == 5 {
			b := view.CommunityCards
			for _, q := range view.Players {
				if q.UUID == hp.UUID {
					cards, score := eval.BestFiveOfSeven(q.Cards[0], q.Cards[1], b[0], b[1], b[2], b[3], b[4])
					p.Shown = true
					p.Hand, p.score = handName(score, cards), score
				}
			}
		}
	}

	tally(t.totals, r)
	t.history = append(t.history, *r)
	if len(t.history) > maxHands {
		t.history = t.history[len(t.history)-maxHands:]
	}

	u := stateUpdate{byUUID: map[string][]byte{}}
	more := len(t.history) > 1
	for _, p := range view.Players {
		u.byUUID[p.UUID] = createHistory([]handRecord{r.viewFor(p.UUID)}, false, more)
	}
	u.spectator = createHistory([]handRecord{r.viewFor("")}, false, more)
	select {
	case t.updates <- u:
	case <-t.done:
	}
	return r
}

// historyFor is up to handsPerPage hands from before hand number before (0
// for the latest), as uuid may see them, and whether there are older ones.
// Caller holds t.mu.
func historyFor(t *table, uuid string, before int) ([]handRecord, bool) {
	end := len(t.history)
	if before > 0 {
		end = 0
		for end < len(t.history) && t.history[end].Hand < before {
			end++
		}
	}
	start := max(end-handsPerPage, 0)
	hands := make([]handRecord, 0, end-start)
	for _, r := range t.history[start:end] {
		hands = append(hands, r.viewFor(uuid))
	}
	return hands, start > 0
}

// handleGetHands sends c the hands before hand number before.
func handleGetHands(c *Client, before int) {
	t := c.table
	t.mu.Lock()
	defer t.mu.Unlock()
	hands, more := historyFor(t, c.id(), before)
	trySend(c, createHistory(hands, false, more))
}

// createHistory sends hands, oldest first. full replaces what the client
// has; more says older hands than the first are kept.
func createHistory(hands []handRecord, full, more bool) []byte {
	resp, err := json.Marshal(handHistory{base{actionHandHistory}, hands, full, more})
	if err != nil {
		slog.Default().Warn("Marshal hand history", "error", err)
	}
	return resp
}
