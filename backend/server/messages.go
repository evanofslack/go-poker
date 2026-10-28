package server

import (
	"encoding/json"

	"github.com/evanofslack/go-poker/table"
)

// The websocket protocol. tygo turns this file into web/interfaces/server.ts;
// run `just types` after changing it.

// inbound (client) actions
const (
	ActionCreateTable = "create-table"
	ActionJoinTable   = "join-table"
	ActionLeaveTable  = "leave-table"
	ActionListTables  = "list-tables" // works before joining a table
	ActionSendMessage = "send-message"
	ActionTakeSeat    = "take-seat"
	ActionStartGame   = "start-game"
	ActionCheck       = "player-check"
	ActionCall        = "player-call"
	ActionRaise       = "player-raise"
	ActionFold        = "player-fold"
	ActionLeaveSeat   = "leave-seat"
	ActionSitOut      = "sit-out"
	ActionRebuy       = "rebuy"
	ActionGetHands    = "get-hands"
	ActionShowCards   = "show-cards" // between hands
	ActionDuckHunt    = "duck-hunt"  // between hands, on tables that allow it
	ActionAddTime     = "add-time"   // on your turn, spends a time card
	ActionStraddle    = "straddle"   // yes answers the straddle ask
	ActionPreset      = "straddle-preset"
	ActionRunIt       = "run-it" // runs is the pick in the run it vote

	// host only
	ActionAddBot            = "add-bot"
	ActionRemoveBot         = "remove-bot"
	ActionMakeHost          = "make-host"
	ActionSetBanker         = "set-banker"
	ActionMoveToSpectator   = "move-to-spectator"
	ActionRemovePlayer      = "remove-player"
	ActionSitOutPlayer      = "sit-out-player"
	ActionApproveBuy        = "approve-buy-in"
	ActionDenyBuy           = "deny-buy-in"
	ActionAdjustChips       = "adjust-chips"
	ActionPauseGame         = "pause-game"
	ActionResumeGame        = "resume-game"
	ActionEndSession        = "end-session"
	ActionStartStandUp      = "start-stand-up" // amount is the payout
	ActionCancelStandUp     = "cancel-stand-up"
	ActionSpecialHand       = "special-hand" // kind: the next hand is a bomb, double or double-bomb
	ActionCancelSpecialHand = "cancel-special-hand"
	ActionUpdateSettings    = "update-settings" // settings has all of them, but the seat count stays
	ActionSetPassword       = "set-password"
)

// Inbound is every message a client sends. Action says which fields are used.
type Inbound struct {
	Action string `json:"action"`

	// create-table and join-table
	Tablename string          `json:"tablename,omitempty"` // also leave-table
	Username  string          `json:"username,omitempty"`
	Token     string          `json:"token,omitempty"`                                     // reclaims a seat after a disconnect
	Password  string          `json:"password,omitempty"`                                  // sets the table's password on create or set-password, else checked
	Settings  json.RawMessage `json:"settings,omitempty" tstype:"Partial<table.Settings>"` // create-table, where missing fields get their default, or update-settings with all of them

	Message string `json:"message,omitempty"` // send-message
	Seat    int    `json:"seat,omitempty"`    // take-seat, add-bot, remove-bot
	Amount  int64  `json:"amount,omitempty"`  // take-seat and add-bot buy-in, rebuy, adjust-chips
	To      uint   `json:"to,omitempty"`      // player-raise: the whole bet this street
	Before  int    `json:"before,omitempty"`  // get-hands: hands before this number
	Name    string `json:"name,omitempty"`    // add-bot, and the host actions that name a player
	Level   string `json:"level,omitempty"`   // add-bot
	Reason  string `json:"reason,omitempty"`  // adjust-chips
	Yes     bool   `json:"yes,omitempty"`     // straddle
	Preset  string `json:"preset,omitempty"`  // straddle-preset: ask, always or never
	Runs    int    `json:"runs,omitempty"`    // run-it
	Kind    string `json:"kind,omitempty"`    // special-hand
}

// outbound (server) actions
const (
	ActionJoined      = "joined-table"
	ActionWelcome     = "welcome"
	ActionTableUpdate = "table-update"
	ActionHandHistory = "hand-history"
	ActionChat        = "chat"
	ActionToken       = "token"
	ActionError       = "error"
	ActionTables      = "tables"
	ActionPassword    = "password"
)

// Joined confirms a create-table or join-table.
type Joined struct {
	Action    string `json:"action"`
	Tablename string `json:"tablename"`
	Username  string `json:"username"`
	ID        string `json:"id"`
	Password  string `json:"password,omitempty"` // only to this client, so it can share invite links
}

// Welcome follows Joined with the table so far.
type Welcome struct {
	Action  string                  `json:"action"`
	View    table.View              `json:"view"`
	Logs    []table.Log             `json:"logs"`
	Stats   []table.HandStats       `json:"stats"`  // every hand the table keeps
	Totals  map[string]table.Totals `json:"totals"` // by name
	History []table.HandRecord      `json:"history"`
	More    bool                    `json:"more"` // the table keeps older hands than History's first
}

// TableUpdate is the table after something changed, and what happened, both
// as the receiver may see them.
type TableUpdate struct {
	Action string       `json:"action"`
	View   table.View   `json:"view"`
	Events []EventEnvel `json:"events"`
}

// EventEnvel is an event with its type: "log", "hand-done", or one of
// holdem's events.
type EventEnvel struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data" tstype:"unknown"`
}

// HandHistory answers get-hands.
type HandHistory struct {
	Action string             `json:"action"`
	Hands  []table.HandRecord `json:"hands"`
	More   bool               `json:"more"`
}

type ChatOut struct {
	Action            string `json:"action"`
	table.ChatMessage `tstype:",extends"`
}

// TokenOut is the secret for reclaiming a seat, only sent to its player.
// Empty when the seat is given up.
type TokenOut struct {
	Action string `json:"action"`
	Token  string `json:"token"`
}

// PasswordOut is the table's new password after the host changed it, for
// invite links. Empty once removed.
type PasswordOut struct {
	Action   string `json:"action"`
	Password string `json:"password"`
}

// ErrorOut tells one client why its action was rejected.
type ErrorOut struct {
	Action  string `json:"action"`
	Code    string `json:"code,omitempty"` // "password-required", "wrong-password", "removed", "closed"
	Message string `json:"message"`
}

// Tables answers list-tables with the public tables, best to join first.
type Tables struct {
	Action string          `json:"action"`
	Tables []table.Summary `json:"tables"`
}
