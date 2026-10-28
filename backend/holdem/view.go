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
	Bomb      uint     `json:"bomb"`
	ToAct     int      `json:"toAct"`
	Board     []Card   `json:"board"`
	Hunted    []Card   `json:"hunted"`
	Runs      int      `json:"runs"`
	RunFrom   int      `json:"runFrom"`
	Boards    [][]Card `json:"boards"`
	Pots      []Pot    `json:"pots"`
	MinRaise  uint     `json:"minRaise"`
	Hand      int      `json:"hand"`
}

// View is the game as the player in seat sees it. Seat 0 is a spectator.
func (s *State) View(seat int) View {
	c := s.Clone()
	for i := range c.Players {
		p := &c.Players[i]
		switch {
		case p.Cards == nil:
			p.Cards = []Card{}
		case p.Seat != seat && !p.Shown:
			p.Cards = make([]Card, len(p.Cards))
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
	boards := c.Boards
	if boards == nil {
		boards = [][]Card{}
	}
	pots := c.Pots
	if pots == nil {
		pots = []Pot{}
	}
	for i := range pots {
		if pots[i].Runs == nil {
			pots[i].Runs = []PotRun{}
		}
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
		Bomb:      c.Bomb,
		ToAct:     c.ToAct,
		Board:     board,
		Hunted:    hunted,
		Runs:      c.Runs,
		RunFrom:   c.RunFrom,
		Boards:    boards,
		Pots:      pots,
		MinRaise:  c.MinRaise,
		Hand:      c.Hands,
	}
}
