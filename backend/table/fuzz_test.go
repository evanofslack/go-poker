package table

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// TestRandomCommands throws random commands from random people at tables,
// with time passing, and checks the table stays consistent.
func TestRandomCommands(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e"}
	var hands, steps, closedEarly int
	defer func() { t.Logf("hands %d steps %d closed early %d", hands, steps, closedEarly) }()
	for run := range 200 {
		r := rand.New(rand.NewPCG(uint64(run), 7))
		set := DefaultSettings
		set.Seats = uint(2 + r.IntN(4))
		set.ApproveRebuys = r.IntN(2) == 0
		set.Ante = r.UintN(3) * set.BigBlind / 4
		set.DuckHunt = r.IntN(2) == 0
		set.SevenDeuce = r.UintN(3) * set.BigBlind
		if r.IntN(3) == 0 {
			set.TurnTime = 0
		}
		h := newHarness(t, set, "")
		joins := 0
		// start most runs with a game going
		h.join("a")
		h.join("b")
		h.sit("a", 1, set.MinBuy)
		h.sit("b", 2, set.MaxBuy)
		if r.IntN(4) > 0 {
			h.ok(Start{Person: h.ids["a"]})
		}

		for step := range 400 {
			if h.closed != nil {
				closedEarly++
				break
			}
			steps++
			name := names[r.IntN(len(names))]
			id := h.ids[name]
			other := names[r.IntN(len(names))]
			var cmd Command
			switch r.IntN(30) {
			case 0, 1:
				joins++
				newID := fmt.Sprintf("%s-%d", name, joins)
				if res := h.do(Join{ID: newID, Name: name, Token: h.tokens[name]}); res.Joined != nil {
					h.ids[name] = res.Joined.ID
				}
			case 2:
				cmd = Leave{Person: id}
			case 3:
				cmd = Disconnect{Person: id}
			case 4, 5:
				cmd = TakeSeat{Person: id, Seat: 1 + r.IntN(int(set.Seats)+1), BuyIn: set.MinBuy + r.UintN(set.MaxBuy-set.MinBuy+500)}
			case 6:
				cmd = Start{Person: id}
			case 7:
				cmd = LeaveSeat{Person: id}
			case 8:
				cmd = SitOut{Person: id}
			case 9:
				cmd = Rebuy{Person: id, Amount: r.UintN(3000)}
			case 10:
				cmd = AddBot{Person: id, Seat: 1 + r.IntN(int(set.Seats)), BuyIn: set.MinBuy, Name: "bot" + other, Level: "easy"}
			case 11:
				cmd = RemoveBot{Person: id, Seat: 1 + r.IntN(int(set.Seats))}
			case 12:
				kinds := []HostKind{MakeHost, SetBanker, MoveToSpectator, RemovePlayer, SitOutPlayer, ApproveBuy, DenyBuy}
				cmd = HostAction{Person: id, Kind: kinds[r.IntN(len(kinds))], Name: other}
			case 13:
				cmd = AdjustChips{Person: id, Name: other, Amount: r.Int64N(2000) - 1000}
			case 14:
				cmd = Pause{Person: id, Pause: r.IntN(2) == 0}
			case 15:
				if r.IntN(10) == 0 {
					cmd = EndSession{Person: id}
				}
			case 16:
				cmd = GetHands{Person: id, Before: r.IntN(5)}
			case 17:
				h.advance(time.Duration(r.IntN(40)) * time.Second)
			case 18:
				cmd = Show{Person: id}
			case 19:
				cmd = Hunt{Person: id}
			default:
				// whoever is to act moves, if they're a person
				g := &h.s.Game
				if !g.Betting {
					h.advance(time.Second)
					break
				}
				seat := h.s.Seats[g.ToAct]
				l := g.Legal(g.ToAct)
				actions := []holdem.Action{holdem.Fold{}, holdem.Call{}, holdem.Check{}, holdem.Raise{To: l.RaiseMin}, holdem.Raise{To: l.RaiseMax}}
				cmd = Play{Person: seat.Person, Action: actions[r.IntN(len(actions))]}
			}
			if cmd != nil {
				func() {
					defer func() {
						if p := recover(); p != nil {
							t.Fatalf("run %d step %d: %T %+v panicked: %v", run, step, cmd, cmd, p)
						}
					}()
					h.do(cmd)
				}()
			}
			h.checkInvariants()
		}
		hands += h.s.Game.Hands
	}
}
