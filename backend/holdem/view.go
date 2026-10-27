package holdem

// View is the game as one player may see it: their own cards and any that
// are face up. It leaves out the deck and seed.
type View struct {
	Config    Config   `json:"config"`
	Players   []Player `json:"players"`
	Street    Street   `json:"street"`
	Betting   bool     `json:"betting"`
	Button    int      `json:"button"`
	SB        int      `json:"sb"`
	BB        int      `json:"bb"`
	Straddles []int    `json:"straddles"`
	ToAct     int      `json:"toAct"`
	Board     []Card   `json:"board"`
	Hunted    []Card   `json:"hunted"`
	Pots      []Pot    `json:"pots"`
	MinRaise  uint     `json:"minRaise"`
	Hand      int      `json:"hand"`
}

// View is the game as the player in seat sees it. Seat 0 is a spectator.
func (s *State) View(seat int) View {
	c := s.Clone()
	for i := range c.Players {
		if p := &c.Players[i]; p.Seat != seat && !p.Shown {
			p.Cards = [2]Card{}
		}
	}
	players := c.Players
	if players == nil {
		players = []Player{}
	}
	board := c.Board
	if board == nil {
		board = []Card{}
	}
	straddles := c.Straddles
	if straddles == nil {
		straddles = []int{}
	}
	hunted := c.Hunted
	if hunted == nil {
		hunted = []Card{}
	}
	pots := c.Pots
	if pots == nil {
		pots = []Pot{}
	}
	return View{
		Config:    c.Config,
		Players:   players,
		Street:    c.Street,
		Betting:   c.Betting,
		Button:    c.Button,
		SB:        c.SB,
		BB:        c.BB,
		Straddles: straddles,
		ToAct:     c.ToAct,
		Board:     board,
		Hunted:    hunted,
		Pots:      pots,
		MinRaise:  c.MinRaise,
		Hand:      c.Hands,
	}
}
