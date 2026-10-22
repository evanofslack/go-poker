package holdem

// Event is something that happened in the game. Everything that changes the
// game returns the events it caused, in order.
type Event interface {
	EventType() string
}

// HandStarted is a new hand being dealt to Seats.
type HandStarted struct {
	Hand   int   `json:"hand"`
	Button int   `json:"button"`
	SB     int   `json:"sb"`
	BB     int   `json:"bb"`
	Seats  []int `json:"seats"`
}

// Dealt is a player's hole cards. Only that player may see it.
type Dealt struct {
	Seat  int     `json:"seat"`
	Cards [2]Card `json:"cards"`
}

// Action kinds
const (
	KindSmallBlind = "small blind"
	KindBigBlind   = "big blind"
	KindCheck      = "check"
	KindCall       = "call"
	KindBet        = "bet"
	KindRaise      = "raise"
	KindFold       = "fold"
)

// Acted is a player posting a blind or acting.
type Acted struct {
	Seat   int    `json:"seat"`
	Street Street `json:"street"`
	Kind   string `json:"kind"`   // one of the Kind constants
	Amount uint   `json:"amount"` // chips put in
	To     uint   `json:"to"`     // their bet this street after
	AllIn  bool   `json:"allIn"`
}

// StreetDealt is the board cards for a new street.
type StreetDealt struct {
	Street Street `json:"street"`
	Cards  []Card `json:"cards"`
}

// Returned is chips nobody called going back to the player who bet them.
type Returned struct {
	Seat   int  `json:"seat"`
	Amount uint `json:"amount"`
}

// Shown is a player's cards turned face up.
type Shown struct {
	Seat  int     `json:"seat"`
	Cards [2]Card `json:"cards"`
}

// PotWon is a pot paid out.
type PotWon struct {
	Pot     int    `json:"pot"` // index in State.Pots
	Amount  uint   `json:"amount"`
	Winners []int  `json:"winners"`
	Hand    string `json:"hand"` // empty without a showdown
}

// HandEnded is the end of a hand, after every pot is paid.
type HandEnded struct {
	Hand int `json:"hand"`
}

func (HandStarted) EventType() string { return "hand-started" }
func (Dealt) EventType() string       { return "dealt" }
func (Acted) EventType() string       { return "acted" }
func (StreetDealt) EventType() string { return "street-dealt" }
func (Returned) EventType() string    { return "returned" }
func (Shown) EventType() string       { return "shown" }
func (PotWon) EventType() string      { return "pot-won" }
func (HandEnded) EventType() string   { return "hand-ended" }

// Visible reports whether the player in seat may see e. Seat 0 is a spectator.
func Visible(e Event, seat int) bool {
	if d, ok := e.(Dealt); ok {
		return d.Seat == seat
	}
	return true
}
