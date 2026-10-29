package table

import (
	"cmp"
	"slices"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// BuyRequest is a rebuy, or a returning player sitting back down, waiting for
// the host on a table that approves rebuys.
type BuyRequest struct {
	Name   string    `json:"name"`
	Amount uint      `json:"amount"`
	Seat   int       `json:"seat"` // the seat asked for, 0 to rebuy at their seat
	At     time.Time `json:"at"`   // when it was asked, for ordering
}

// PlayerView is a seat as everyone sees it.
type PlayerView struct {
	holdem.Player `tstype:",extends"`
	Name          string `json:"name"`
	Bot           string `json:"bot"`          // level, empty for a person
	Disconnected  bool   `json:"disconnected"` // may still reconnect
	Leaving       bool   `json:"leaving"`
}

// View is the table as one person sees it.
type View struct {
	Players   []PlayerView    `json:"players"`
	Street    holdem.Street   `json:"street"`
	Betting   bool            `json:"betting"`
	Button    int             `json:"button"`
	SB        int             `json:"sb"`
	BB        int             `json:"bb"`
	Straddles []int           `json:"straddles"` // seats that straddled this hand, in order
	ToAct     int             `json:"toAct"`
	Board     []holdem.Card   `json:"board"`
	Hunted    []holdem.Card   `json:"hunted"`  // the rest of the board, from a duck hunt
	Runs      int             `json:"runs"`    // times the board runs out, 0 until decided
	RunFrom   int             `json:"runFrom"` // board cards dealt before the runs split
	Boards    [][]holdem.Card `json:"boards"`  // every finished runout, the first being Board; on double board both, from the flop
	Bomb      uint            `json:"bomb"`    // this hand is a bomb pot of this much each, 0 if not
	Double    bool            `json:"double"`  // this hand is double board
	Pots      []holdem.Pot    `json:"pots"`
	MinRaise  uint            `json:"minRaise"`
	Hand      int             `json:"hand"`

	You        int          `json:"you"`        // your seat, 0 when watching
	Legal      holdem.Legal `json:"legal"`      // what you can do now
	SitOutNext bool         `json:"sitOutNext"` // you asked to sit out once this hand ends
	ShowClaims []string     `json:"showClaims"` // what showing your cards now would win you, like "7-2" or "stand-up"

	Settings       Settings       `json:"settings"`
	Next           *Settings      `json:"next"` // settings from the next hand, if the host changed them mid-hand
	HasPassword    bool           `json:"hasPassword"`
	Host           string         `json:"host"`
	Running        bool           `json:"running"`
	Paused         bool           `json:"paused"`
	Ending         bool           `json:"ending"`   // the session ends after this hand
	Ended          bool           `json:"ended"`    // the session is over and the ledger final
	ClosesIn       int64          `json:"closesIn"` // ms until an ended table closes
	TurnEndsIn     int64          `json:"turnEndsIn"`
	BuyRequests    []BuyRequest   `json:"buyRequests"` // oldest first
	Watchers       []string       `json:"watchers"`    // connected without a seat
	Ledger         LedgerView     `json:"ledger"`
	Acts           map[int]string `json:"acts"` // what each seat did this street
	StandUp        *StandUp       `json:"standUp"`
	StraddleAsk    *StraddleAsk   `json:"straddleAsk"`
	StraddleEndsIn int64          `json:"straddleEndsIn"` // ms the player asked to straddle has left
	StraddlePreset string         `json:"straddlePreset"` // yours: ask, always or never
	RunVote        *RunVote       `json:"runVote"`
	RunVoteEndsIn  int64          `json:"runVoteEndsIn"` // ms the run it vote has left
	LastStandUp    uint           `json:"lastStandUp"`   // the payout of the stand-up round that just ended
	SpecialNext    string         `json:"specialNext"`   // the next hand's special, if any: bomb, double or double-bomb
	SpecialIn      int            `json:"specialIn"`     // hands until the schedule's next special hand, 0 with no schedule
	MadeHand       []string       `json:"madeHand"`      // yours so far, like "two pair", on each board; empty when you're not in a hand
}

// ViewFor is the table as the person with id sees it, at now.
func ViewFor(s *State, id string, now time.Time) View {
	var you int
	if p := s.People[id]; p != nil {
		you = p.Seat
	}
	g := s.Game.View(you)
	v := View{
		Street:         g.Street,
		Betting:        g.Betting,
		Button:         g.Button,
		SB:             g.SB,
		BB:             g.BB,
		Straddles:      g.Straddles,
		ToAct:          g.ToAct,
		Board:          g.Board,
		Hunted:         g.Hunted,
		Runs:           g.Runs,
		RunFrom:        g.RunFrom,
		Boards:         g.Boards,
		Bomb:           g.Bomb,
		Double:         g.Double,
		Pots:           g.Pots,
		MinRaise:       g.MinRaise,
		Hand:           g.Hand,
		You:            you,
		Legal:          s.Game.Legal(you),
		Settings:       s.Settings,
		Next:           s.Next,
		HasPassword:    s.Password != "",
		Host:           s.Host,
		Running:        s.Running,
		Paused:         s.Paused,
		Ending:         s.Ending,
		Ended:          s.Ended(),
		ClosesIn:       until(s.Timers, TimerClose, now),
		TurnEndsIn:     until(s.Timers, TimerTurn, now),
		BuyRequests:    BuyRequests(s),
		Watchers:       Watchers(s),
		Ledger:         Ledger(s),
		Acts:           shownActs(s),
		StandUp:        s.StandUp,
		StraddleAsk:    s.StraddleAsk,
		StraddleEndsIn: until(s.Timers, TimerStraddle, now),
		RunVote:        s.RunVote,
		RunVoteEndsIn:  until(s.Timers, TimerRunVote, now),
		LastStandUp:    s.LastStandUp,
		SpecialNext:    s.SpecialNext,
		SpecialIn:      s.specialIn(),
		MadeHand:       madeHand(s, you),
	}
	v.ShowClaims = ShowClaims(s, you)
	if you != 0 {
		v.SitOutNext = s.Seats[you].SitOutNext
		v.StraddlePreset = cmp.Or(s.Seats[you].Straddle, PresetAsk)
	}
	for _, p := range g.Players {
		seat := s.Seats[p.Seat]
		person := s.People[seat.Person]
		v.Players = append(v.Players, PlayerView{
			Player:       p,
			Name:         seat.Name,
			Bot:          seat.Bot,
			Disconnected: person != nil && !person.Connected,
			Leaving:      seat.Leaving,
		})
	}
	if v.Players == nil {
		v.Players = []PlayerView{}
	}
	return v
}

// until is how many ms are left on timer, 0 if it isn't set.
func until(timers map[Timer]time.Time, timer Timer, now time.Time) int64 {
	at, ok := timers[timer]
	if !ok {
		return 0
	}
	return max(at.Sub(now).Milliseconds(), 0)
}

// Watchers are the names of the connected people without a seat, sorted.
func Watchers(s *State) []string {
	names := []string{}
	for _, p := range s.People {
		if p.Seat == 0 && p.Connected {
			names = append(names, p.Name)
		}
	}
	slices.Sort(names)
	return names
}

var streetOrder = map[holdem.Street]int{holdem.Preflop: 1, holdem.Flop: 2, holdem.Turn: 3, holdem.River: 4}

// madeHand is the hand seat makes so far, for their own hint: empty unless
// they're in a hand being played.
func madeHand(s *State, seat int) []string {
	p := s.Game.Player(seat)
	if p == nil || !p.In || !s.Game.InHand() {
		return []string{}
	}
	if s.Game.Double && len(s.Game.Boards) == 2 {
		return []string{s.Game.Config.Game.MadeHand(p.Cards, s.Game.Boards[0]), s.Game.Config.Game.MadeHand(p.Cards, s.Game.Boards[1])}
	}
	return []string{s.Game.Config.Game.MadeHand(p.Cards, s.Game.Board)}
}

// shownActs are the actions the seats show: this street's. Between hands the
// last street's stay up.
func shownActs(s *State) map[int]string {
	street := s.Game.Street
	if street == holdem.NoStreet {
		for _, a := range s.Acts {
			if streetOrder[a.Street] > streetOrder[street] {
				street = a.Street
			}
		}
	}
	shown := map[int]string{}
	for seat, a := range s.Acts {
		if a.Street == street {
			shown[seat] = a.Label
		}
	}
	return shown
}

// EventsFor is events as the person with id may see them.
func EventsFor(s *State, events []Event, id string) []Event {
	var seat int
	var name string
	if p := s.People[id]; p != nil {
		seat, name = p.Seat, p.Name
	}
	out := []Event{}
	for _, e := range events {
		switch d := e.(type) {
		case HandDone:
			d.Record = d.Record.viewFor(name)
			e = d
		case HandUpdated:
			d.Record = d.Record.viewFor(name)
			e = d
		}
		if holdem.Visible(e, seat) {
			out = append(out, e)
		}
	}
	return out
}

// Summary is what the home page shows of a public table.
type Summary struct {
	Name        string `json:"name"`
	SmallBlind  uint   `json:"sb"`
	BigBlind    uint   `json:"bb"`
	Ante        uint   `json:"ante"`
	MinBuy      uint   `json:"minBuy"`
	MaxBuy      uint   `json:"maxBuy"`
	Seats       uint   `json:"seats"`
	Players     uint   `json:"players"`  // seated, bots included
	Bots        uint   `json:"bots"`     // seated bots
	Watchers    uint   `json:"watchers"` // connected without a seat
	Running     bool   `json:"running"`
	Paused      bool   `json:"paused"`
	HasPassword bool   `json:"hasPassword"`
}

// Summarize is s for the home page.
func Summarize(s *State) Summary {
	sum := Summary{
		Name:        s.Name,
		SmallBlind:  s.Settings.SmallBlind,
		BigBlind:    s.Settings.BigBlind,
		Ante:        s.Settings.Ante,
		MinBuy:      s.Settings.MinBuy,
		MaxBuy:      s.Settings.MaxBuy,
		Seats:       s.Settings.Seats,
		Watchers:    uint(len(Watchers(s))),
		Running:     s.Running,
		Paused:      s.Paused,
		HasPassword: s.Password != "",
	}
	for _, seat := range s.Seats {
		if seat.Leaving {
			continue
		}
		sum.Players++
		if seat.Bot != "" {
			sum.Bots++
		}
	}
	return sum
}
