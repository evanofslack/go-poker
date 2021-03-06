package server

import (
	"github.com/evanofslack/go-poker/poker"
)

// inbound (client) actions
const (
	actionJoinTable   string = "join-table"
	actionLeaveTable  string = "leave-table"
	actionSendMessage string = "send-message"
	actionSendLog     string = "send-log"
	actionNewPlayer   string = "new-player"
	actionTakeSeat    string = "take-seat"
	actionStartGame   string = "start-game"
	actionDealGame    string = "deal-game"
	actionResetGame   string = "reset-game"
	actionPlayerCall  string = "player-call"
	actionPlayerCheck string = "player-check"
	actionPlayerRaise string = "player-raise"
	actionPlayerFold  string = "player-fold"
)

type base struct {
	// allows for correctly identifying messages
	Action string `json:"action"`
}

type joinTable struct {
	base             // actionJoinTable
	Tablename string `json:"tablename"`
}

type leaveTable struct {
	base             // actionLeaveTable
	Tablename string `json:"tablename"`
}

type sendMessage struct {
	base            // actionSendMessage
	Username string `json:"username"`
	Message  string `json:"message"`
}

type sendLog struct {
	base           // actionSendLog
	Message string `json:"message"`
}

type newPlayer struct {
	base            // actionNewPlayer
	Username string `json:"username"`
}

type takeSeat struct {
	base            // actionTakeSeat
	Username string `json:"username"`
	SeatID   uint   `json:"seatID"`
	BuyIn    uint   `json:"buyIn"`
}

type startGame struct {
	base // actionStartGame
}

type resetGame struct {
	base // actionResetGame
}

type dealGame struct {
	base // actionDealGame
}

type playerCall struct {
	base // actionPlayerCall
}

type playerCheck struct {
	base // actionPlayerCheck
}

type playerRaise struct {
	base        // actionPlayerRaise
	Amount uint `json:"amount"`
}

type playerFold struct {
	base // actionPlayerFold
}

// outbound (server) actions
const (
	actionNewMessage       string = "new-message"
	actionNewLog           string = "new-log"
	actionUpdateGame       string = "update-game"
	actionUpdatePlayerUUID string = "update-player-uuid"
)

type newMessage struct {
	base             // actionNewMessage
	Id        string `json:"uuid"`
	Message   string `json:"message"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type newLog struct {
	base             // actionNewLog
	Id        string `json:"uuid"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type updateGame struct {
	base                 // actionUpdateGame
	Game *poker.GameView `json:"game"`
}

type updatePlayerUUID struct {
	base        //actionUpdatePlayerUUID
	Uuid string `json:"uuid"`
}
