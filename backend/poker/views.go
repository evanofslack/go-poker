package poker

import (
	"github.com/alexclewontin/riverboat/eval"
)

// GameView is the type that represents a snapshot of a Game's state.
type GameView struct {
	Running        bool        `json:"running"`
	DealerNum      uint        `json:"dealer"`
	ActionNum      uint        `json:"action"`
	UTGNum         uint        `json:"utg"`
	SBNum          uint        `json:"sb"`
	BBNum          uint        `json:"bb"`
	CommunityCards []eval.Card `json:"communityCards"`
	Stage          GameStage   `json:"stage"`
	Betting        bool        `json:"betting"`
	Config         GameConfig  `json:"config"`
	Players        []player    `json:"players"`
	Deck           eval.Deck   `json:"-"`
	Pots           []Pot       `json:"pots"`
	MinRaise       uint        `json:"minRaise"`
	ReadyCount     uint        `json:"readyCount"`
}

func cardReader(cards []eval.Card) []string {
	var readable []string
	for _, c := range cards {
		readable = append(readable, c.String())
	}
	return readable
}

func (g *Game) copyToView() *GameView {
	//TODO: Is there some way to do this programatically? I considered using
	// reflection, but since that happens at runtime it is less performant.
	// Something like reflection, but evaluated at compile-time would be ideal
	// Probably using go generate.

	//WARNING: This needs to be the deepest of deep copies. If adding a field,
	//make sure that it is. An example: copying a slice of structs, where the struct
	//has a field that is a slice: this doesn't work by default. Write a helper function.
	view := &GameView{
		Running:        g.running,
		DealerNum:      g.dealerNum,
		ActionNum:      g.actionNum,
		UTGNum:         g.utgNum,
		SBNum:          g.sbNum,
		BBNum:          g.bbNum,
		CommunityCards: append([]eval.Card{}, g.communityCards...),
		Stage:          g.getStage(),
		Betting:        g.getBetting(),
		Config:         g.config,
		Players:        append([]player{}, g.players...),
		Deck:           append([]eval.Card{}, g.deck...),
		Pots:           copyPots(g.pots),
		MinRaise:       g.minRaise,
		ReadyCount:     g.readyCount(),
	}

	return view
}

func copyPots(src []Pot) []Pot {
	ret := make([]Pot, len(src))
	for i := range src {
		ret[i].Amt = src[i].Amt
		ret[i].TopShare = src[i].TopShare
		ret[i].WinningScore = src[i].WinningScore
		ret[i].EligiblePlayerNums = append([]uint{}, src[i].EligiblePlayerNums...)
		ret[i].WinningPlayerNums = append([]uint{}, src[i].WinningPlayerNums...)
		ret[i].WinningHand = append([]eval.Card{}, src[i].WinningHand...)
	}

	return ret
}

// FillFromView is primarily for loading a stored view from a persistence layer
func (g *Game) FillFromView(gv *GameView) {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	g.running = gv.Running
	g.dealerNum = gv.DealerNum
	g.actionNum = gv.ActionNum
	g.utgNum = gv.UTGNum
	g.bbNum = gv.BBNum
	g.sbNum = gv.SBNum
	g.communityCards = append([]eval.Card{}, gv.CommunityCards...)
	g.setStageAndBetting(gv.Stage, gv.Betting)
	g.config = gv.Config
	g.players = append([]player{}, gv.Players...)
	g.deck = append([]eval.Card{}, gv.Deck...)
	g.pots = copyPots(gv.Pots)
	g.minRaise = gv.MinRaise
}

// GeneratePlayerView is primarily for creating a view that can be serialized for delivery to a specific player
// The generated view holds only the information that the player denoted by pn is entitled to see at the moment it is generated.
func (g *Game) GeneratePlayerView(pn uint) *GameView {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	gv := g.copyToView()
	gv.Deck = nil

	// D. R. Y.!
	hideCards := func(pn2 uint) { gv.Players[pn2].Cards = [2]eval.Card{0, 0} }
	showCards := func(pn2 uint) { gv.Players[pn2].Cards = [2]eval.Card{g.players[pn2].Cards[0], g.players[pn2].Cards[1]} }

	allInCount := 0
	inCount := 0

	for i, p := range g.players {
		if uint(i) != pn {
			hideCards(uint(i))
		} else {
			if !g.players[pn].In {
				hideCards(uint(i))
			}
		}

		if p.allIn() {
			allInCount++
		}

		if p.In {
			inCount++
		}

	}

	// If in a heads-up situation
	if allInCount == inCount {
		for i, p := range g.players {
			if p.In {
				showCards(uint(i))
			}
		}
	}

	if g.getStage() == PreDeal && inCount > 1 {

		showCards(g.calledNum)
		_, scoreToBeat := eval.BestFiveOfSeven(
			g.players[g.calledNum].Cards[0],
			g.players[g.calledNum].Cards[1],
			g.communityCards[0],
			g.communityCards[1],
			g.communityCards[2],
			g.communityCards[3],
			g.communityCards[4],
		)

		for i := range g.players {
			pni := (g.calledNum + uint(i)) % uint(len(g.players))
			_, iScore := eval.BestFiveOfSeven(
				g.players[pni].Cards[0],
				g.players[pni].Cards[1],
				g.communityCards[0],
				g.communityCards[1],
				g.communityCards[2],
				g.communityCards[3],
				g.communityCards[4],
			)

			if (iScore <= scoreToBeat) && g.players[pni].In {
				showCards(pni)
				scoreToBeat = iScore
			}
		}

		for _, pot := range g.pots {

			for _, j := range pot.WinningPlayerNums {
				showCards(j)
			}
		}
	}

	return gv
}

// GenerateOmniView is primarily for creating a view that can be serialized for delivery to a persistance layer, like a db or in-memory store
// Nothing is censored, not even the contents of the deck
func (g *Game) GenerateOmniView() *GameView {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	return g.copyToView()

}
