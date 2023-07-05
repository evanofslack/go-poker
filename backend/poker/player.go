package poker

import (
	. "github.com/alexclewontin/riverboat/eval"
	"github.com/google/uuid"
)

type player struct {
	Username   string  `json:"username"`
	UUID       string  `json:"uuid"`
	Position   uint    `json:"position"`
	SeatID     uint    `json:"seatID"`
	Ready      bool    `json:"ready"`
	In         bool    `json:"in"`
	Called     bool    `json:"called"`
	Left       bool    `json:"left"`
	TotalBuyIn uint    `json:"totalBuyIn"`
	Stack      uint    `json:"stack"`
	Bet        uint    `json:"bet"`
	TotalBet   uint    `json:"totalBet"`
	Cards      [2]Card `json:"cards"`
}

func (p *player) allIn() bool {
	return p.In && (p.Stack == 0)
}

func (p *player) initialize() {
	*p = player{}

	p.UUID = uuid.New().String()
	p.Ready = false
	p.In = false
	p.Called = false

}

//putInChips is simply a helper function that transfers the amounts between fields
func (p *player) putInChips(amt uint) {
	if p.Stack > amt {
		p.Bet += amt
		p.TotalBet += amt
		p.Stack -= amt
	} else {
		p.Bet += p.Stack
		p.TotalBet += p.Stack
		p.Stack = 0
	}
}

func (p *player) returnChips(amt uint) {
	if p.TotalBet > amt {
		p.TotalBet -= amt
		p.Stack += amt
	} else {
		p.Stack += p.TotalBet
		p.TotalBet = 0
	}
}
