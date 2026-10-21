package server

import (
	"encoding/json"

	"github.com/evanofslack/go-poker/poker"
)

// inbound (client) actions
const (
	actionCreateTable string = "create-table"
	actionJoinTable   string = "join-table"
	actionLeaveTable  string = "leave-table"
	actionSendMessage string = "send-message"
	actionTakeSeat    string = "take-seat"
	actionStartGame   string = "start-game"
	actionPlayerCall  string = "player-call"
	actionPlayerCheck string = "player-check"
	actionPlayerRaise string = "player-raise"
	actionPlayerFold  string = "player-fold"
	actionLeaveSeat   string = "leave-seat"
	actionSitOut      string = "sit-out"
	actionRebuy       string = "rebuy"
	actionAddBot      string = "add-bot"
	actionRemoveBot   string = "remove-bot"

	// host only
	actionMakeHost        string = "make-host"
	actionSetBanker       string = "set-banker"
	actionMoveToSpectator string = "move-to-spectator"
	actionRemovePlayer    string = "remove-player"
	actionSitOutPlayer    string = "sit-out-player"
	actionAdjustChips     string = "adjust-chips"
	actionPauseGame       string = "pause-game"
	actionResumeGame      string = "resume-game"
	actionEndSession      string = "end-session"
)

type base struct {
	// allows for correctly identifying messages
	Action string `json:"action"`
}

// joinTable is used for both actionCreateTable and actionJoinTable
type joinTable struct {
	base             // actionCreateTable, actionJoinTable
	Tablename string `json:"tablename"`
	Username  string `json:"username"`
	Token     string `json:"token"`    // optional, reclaims a seat after a disconnect
	Password  string `json:"password"` // sets the table's password on create, else checked

	// create-table only, a tableSettings. Missing fields get their default.
	Settings json.RawMessage `json:"settings"`
}

type leaveTable struct {
	base             // actionLeaveTable
	Tablename string `json:"tablename"`
}

type sendMessage struct {
	base            // actionSendMessage
	Username string `json:"username"` // ignored, the server uses the client's username
	Message  string `json:"message"`
}

type takeSeat struct {
	base            // actionTakeSeat
	Username string `json:"username"` // ignored, the server uses the client's username
	SeatID   uint   `json:"seatID"`
	BuyIn    uint   `json:"buyIn"`
}

type startGame struct {
	base // actionStartGame
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

type leaveSeatMsg struct {
	base // actionLeaveSeat
}

// sitOut toggles sitting out: right away between hands, else after the hand
type sitOut struct {
	base // actionSitOut
}

type rebuy struct {
	base        // actionRebuy
	Amount uint `json:"amount"`
}

// addBot is the host sitting a bot down
type addBot struct {
	base          // actionAddBot
	SeatID uint   `json:"seatID"`
	BuyIn  uint   `json:"buyIn"`
	Name   string `json:"name"`
	Level  string `json:"level"`
}

// removeBot is the host taking the bot in a seat out of the game
type removeBot struct {
	base        // actionRemoveBot
	SeatID uint `json:"seatID"`
}

// playerTarget names who a host action is for: actionMakeHost,
// actionSetBanker, actionMoveToSpectator, actionRemovePlayer, actionSitOutPlayer
type playerTarget struct {
	base
	Name string `json:"name"`
}

type adjustChips struct {
	base          // actionAdjustChips
	Name   string `json:"name"`
	Amount int64  `json:"amount"` // chips to add, or take away when negative
	Reason string `json:"reason"` // optional, shown in the log
}

// pauseGame, resumeGame and endSession carry nothing: actionPauseGame,
// actionResumeGame, actionEndSession
type sessionAction struct {
	base
}

// outbound (server) actions
const (
	actionNewMessage       string = "new-message"
	actionNewLog           string = "new-log"
	actionUpdateGame       string = "update-game"
	actionUpdatePlayerUUID string = "update-player-uuid"
	actionError            string = "error"
	actionJoinedTable      string = "joined-table"
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
	Host string          `json:"host"` // name of the host

	// milliseconds until the player to act times out, 0 when nobody is betting
	TurnEndsIn int64 `json:"turnEndsIn"`

	// whether this recipient asked to sit out once the current hand ends
	SitOutNext bool `json:"sitOutNext"`

	Settings    tableSettings `json:"settings"`
	HasPassword bool          `json:"hasPassword"` // never the password itself

	Paused   bool  `json:"paused"`   // no new hands are dealt
	Ending   bool  `json:"ending"`   // the session ends after this hand
	Ended    bool  `json:"ended"`    // the session is over and the ledger final
	ClosesIn int64 `json:"closesIn"` // milliseconds until an ended table closes

	// names of the clients at the table without a seat
	Watchers []string `json:"watchers"`

	// names of seated players who dropped and may still reconnect
	Disconnected []string `json:"disconnected"`

	Bots map[string]string `json:"bots"` // level of each bot, by uuid

	Ledger ledgerView `json:"ledger"`
}

// joinedTable confirms a create-table or join-table
type joinedTable struct {
	base             // actionJoinedTable
	Tablename string `json:"tablename"`
	Username  string `json:"username"`
	Uuid      string `json:"uuid"`
	Password  string `json:"password,omitempty"` // only to this client, so it can share invite links
}

// errorMessage tells one client why its action was rejected
type errorMessage struct {
	base             // actionError
	Code      string `json:"code,omitempty"` // for errors the client handles, like errorCodes
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type updatePlayerUUID struct {
	base         //actionUpdatePlayerUUID
	Uuid  string `json:"uuid"`
	Token string `json:"token"` // secret for reclaiming the seat, only sent to its player
}
