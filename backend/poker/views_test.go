package poker

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	. "github.com/alexclewontin/riverboat/eval"
)

func TestGame_GenerateOmniView_Driver(t *testing.T) {

	g := &Game{}
	t.Run("Test Functionality", func(t *testing.T) { g.GenerateOmniView() })
}

func TestGame_GenerateOmniView(t *testing.T) {
	type fields struct {
		mtx            *sync.Mutex
		DealerNum      uint
		ActionNum      uint
		UTGNum         uint
		SBNum          uint
		BBNum          uint
		CommunityCards []Card
		Flags          gameFlags
		Config         GameConfig
		Players        []player
		Deck           Deck
		Pots           []Pot
		MinRaise       uint
	}
	tests := []struct {
		name   string
		fields fields
		want   *GameView
	}{
		{
			name: "Basic (empty structs and slices)",
			fields: fields{
				mtx:       &sync.Mutex{},
				DealerNum: 3,
				ActionNum: 4,
				UTGNum:    5,
				SBNum:     6,
				BBNum:     7,
				CommunityCards: []Card{
					8394515,
					16783383,
					33564957,
					67115551,
					134224677,
				},
				Flags:    9,
				Config:   GameConfig{},
				Players:  []player{},
				Deck:     DefaultDeck,
				Pots:     []Pot{},
				MinRaise: 25,
			},
			want: &GameView{
				DealerNum: 3,
				ActionNum: 4,
				UTGNum:    5,
				SBNum:     6,
				BBNum:     7,
				CommunityCards: []Card{
					8394515,
					16783383,
					33564957,
					67115551,
					134224677,
				},
				Stage:    PreDeal,
				Betting:  true,
				Config:   GameConfig{},
				Players:  []player{},
				Deck:     DefaultDeck,
				Pots:     []Pot{},
				MinRaise: 25,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Game{
				//ignore warning here (the only important functionality is that the mutex is ignored during the deep copy)
				mtx:            *tt.fields.mtx,
				dealerNum:      tt.fields.DealerNum,
				actionNum:      tt.fields.ActionNum,
				utgNum:         tt.fields.UTGNum,
				sbNum:          tt.fields.SBNum,
				bbNum:          tt.fields.BBNum,
				communityCards: tt.fields.CommunityCards,
				flags:          tt.fields.Flags,
				config:         tt.fields.Config,
				players:        tt.fields.Players,
				deck:           tt.fields.Deck,
				pots:           tt.fields.Pots,
				minRaise:       tt.fields.MinRaise,
			}
			got := g.GenerateOmniView()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Game.GenerateOmniView() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestGame_GenerateOmniViewChangedVals(t *testing.T) {
	tests := []struct {
		name   string
		fields *Game
		want   *Game
	}{
		{
			name: "Copy",
			fields: &Game{
				mtx:       sync.Mutex{},
				dealerNum: 3,
				actionNum: 4,
				utgNum:    5,
				sbNum:     6,
				bbNum:     7,
				communityCards: []Card{
					8394515,
					16783383,
					33564957,
					67115551,
					134224677,
				},
				flags: 0xF0,
				config: GameConfig{
					BigBlind:   25,
					SmallBlind: 10,
				},
				players: []player{
					{
						Ready:      true,
						In:         false,
						TotalBuyIn: 100,
						Stack:      105,
						Bet:        10,
						TotalBet:   20,
						Cards: [2]Card{
							33564957,
							67115551,
						},
					},
				},
				deck: DefaultDeck,
				pots: []Pot{
					{
						TopShare:           100,
						Amt:                1000,
						EligiblePlayerNums: []uint{0, 1, 2, 3},
						WinningPlayerNums:  []uint{2},
						WinningHand: []Card{
							MustParseCardString("AS"),
							MustParseCardString("KS"),
							MustParseCardString("QS"),
							MustParseCardString("JS"),
							MustParseCardString("TS"),
						},
						WinningScore: 1,
					},
				},
				minRaise: 25,
			},
			want: &Game{
				dealerNum: 3,
				actionNum: 4,
				utgNum:    5,
				sbNum:     6,
				bbNum:     7,
				communityCards: []Card{
					8394515,
					16783383,
					33564957,
					67115551,
					134224677,
				},
				flags: 0xF0,
				config: GameConfig{
					BigBlind:   25,
					SmallBlind: 10,
				},
				players: []player{
					{
						Ready:      true,
						In:         false,
						TotalBuyIn: 100,
						Stack:      105,
						Bet:        10,
						TotalBet:   20,
						Cards: [2]Card{
							33564957,
							67115551,
						},
					},
				},
				deck: DefaultDeck,
				pots: []Pot{
					{
						TopShare:           100,
						Amt:                1000,
						EligiblePlayerNums: []uint{0, 1, 2, 3},
						WinningPlayerNums:  []uint{2},
						WinningHand: []Card{
							MustParseCardString("AS"),
							MustParseCardString("KS"),
							MustParseCardString("QS"),
							MustParseCardString("JS"),
							MustParseCardString("TS"),
						},
						WinningScore: 1,
					},
				},
				minRaise: 25,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.fields
			view := g.GenerateOmniView()

			view.DealerNum = view.DealerNum + 1
			view.ActionNum = view.ActionNum + 1
			view.UTGNum = view.UTGNum + 1
			view.BBNum = view.BBNum + 1
			view.Betting = !view.Betting
			view.Stage = view.Stage + 1
			view.CommunityCards[2] = 0
			view.Config.BigBlind = view.Config.BigBlind + 2
			view.Players[0].Cards[0] = 0
			view.Players[0].Bet = view.Players[0].Bet + 5
			view.Deck.Pop()
			view.Pots[0].Amt = view.Pots[0].Amt + 1
			view.Pots[0].EligiblePlayerNums[0] = view.Pots[0].EligiblePlayerNums[0] + 5

			jsonView, err := json.Marshal(view)

			if err != nil {
				panic(err)
			}

			t.Logf("%s", string(jsonView))

			if !reflect.DeepEqual(g, tt.want) {
				t.Errorf("\nBefore: %+v\nAfter %+v", tt.want, g)
			}
		})
	}
}

func (g *Game) String() string {
	formatStr := "&{dealerNum:%d actionNum:%d utgNum:%d sbNum:%d bbNum:%d communityCards:%v "
	formatStr += "stage:%v betting:%v config:%+v players:%+v deck:%v pots:%+v minRaise:%v"
	return fmt.Sprintf(formatStr,
		g.dealerNum,
		g.actionNum,
		g.utgNum,
		g.sbNum,
		g.bbNum,
		g.communityCards,
		g.getStage(),
		g.getBetting(),
		g.config,
		g.players,
		g.deck,
		g.pots,
		g.minRaise,
	)
}
