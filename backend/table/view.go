package table

import (
	"slices"
	"time"

	"github.com/evanofslack/go-poker/holdem"
)

// PlayerView is a seat as everyone sees it.
type PlayerView struct {
	holdem.Player
	Name         string `json:"name"`
	Bot          string `json:"bot"`          // level, empty for a person
	Disconnected bool   `json:"disconnected"` // may still reconnect
	Leaving      bool   `json:"leaving"`
}

// View is the table as one person sees it.
type View struct {
	Players  []PlayerView  `json:"players"`
	Street   holdem.Street `json:"street"`
	Betting  bool          `json:"betting"`
	Button   int           `json:"button"`
	SB       int           `json:"sb"`
	BB       int           `json:"bb"`
	ToAct    int           `json:"toAct"`
	Board    []holdem.Card `json:"board"`
	Pots     []holdem.Pot  `json:"pots"`
	MinRaise uint          `json:"minRaise"`
	Hand     int           `json:"hand"`

	You        int          `json:"you"`        // your seat, 0 when watching
	Legal      holdem.Legal `json:"legal"`      // what you can do now
	SitOutNext bool         `json:"sitOutNext"` // you asked to sit out once this hand ends

	Settings    Settings       `json:"settings"`
	HasPassword bool           `json:"hasPassword"`
	Host        string         `json:"host"`
	Running     bool           `json:"running"`
	Paused      bool           `json:"paused"`
	Ending      bool           `json:"ending"`   // the session ends after this hand
	Ended       bool           `json:"ended"`    // the session is over and the ledger final
	ClosesIn    int64          `json:"closesIn"` // ms until an ended table closes
	TurnEndsIn  int64          `json:"turnEndsIn"`
	BuyRequests []BuyRequest   `json:"buyRequests"` // oldest first
	Watchers    []string       `json:"watchers"`    // connected without a seat
	Ledger      LedgerView     `json:"ledger"`
	Acts        map[int]string `json:"acts"` // what each seat did this street
}

// ViewFor is the table as the person with id sees it, at now.
func ViewFor(s *State, id string, now time.Time) View {
	var you int
	if p := s.People[id]; p != nil {
		you = p.Seat
	}
	g := s.Game.View(you)
	v := View{
		Street:      g.Street,
		Betting:     g.Betting,
		Button:      g.Button,
		SB:          g.SB,
		BB:          g.BB,
		ToAct:       g.ToAct,
		Board:       g.Board,
		Pots:        g.Pots,
		MinRaise:    g.MinRaise,
		Hand:        g.Hand,
		You:         you,
		Legal:       s.Game.Legal(you),
		Settings:    s.Settings,
		HasPassword: s.Password != "",
		Host:        s.Host,
		Running:     s.Running,
		Paused:      s.Paused,
		Ending:      s.Ending,
		Ended:       s.Ended(),
		ClosesIn:    until(s.Timers, TimerClose, now),
		TurnEndsIn:  until(s.Timers, TimerTurn, now),
		BuyRequests: BuyRequests(s),
		Watchers:    Watchers(s),
		Ledger:      Ledger(s),
		Acts:        shownActs(s),
	}
	if you != 0 {
		v.SitOutNext = s.Seats[you].SitOutNext
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
		if d, ok := e.(HandDone); ok {
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
