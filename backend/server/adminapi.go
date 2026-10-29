package server

import (
	"encoding/json"

	"github.com/evanofslack/go-poker/table"
)

// The admin API's responses, served on ADMIN_ADDR only. They carry IPs, so
// they never go to players.

// AdminOverview is the server and every open table.
type AdminOverview struct {
	Started     int64           `json:"started"` // unix ms
	Store       string          `json:"store"`   // "memory", "sqlite" or "postgres"
	Connections int             `json:"connections"`
	ConnsByIP   map[string]int  `json:"connsByIp"`
	Tables      []AdminTableRow `json:"tables"`
}

// AdminTableRow is a table in the overview.
type AdminTableRow struct {
	ID          string        `json:"id"`
	Summary     table.Summary `json:"summary"`
	CreatorIP   string        `json:"creatorIp"`
	Public      bool          `json:"public"`
	Ended       bool          `json:"ended"`
	Connections int           `json:"connections"`
}

// AdminTable is one table in full, as a watcher sees it plus who is behind
// each name.
type AdminTable struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	CreatorIP string              `json:"creatorIp"`
	Host      string              `json:"host"`
	Banker    string              `json:"banker"`
	Settings  table.Settings      `json:"settings"`
	View      table.View          `json:"view"`
	People    []AdminPerson       `json:"people"`
	Ledger    []table.LedgerEntry `json:"ledger"`
	Logs      []table.Log         `json:"logs"` // the last adminLogs
}

// AdminPerson is someone at a table and their connection.
type AdminPerson struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Seat      int    `json:"seat"` // 0 when watching
	Connected bool   `json:"connected"`
	IP        string `json:"ip"`    // empty while disconnected
	Conn      string `json:"conn"`  // the connection's id in the audit
	Since     int64  `json:"since"` // unix ms the connection opened, 0 while disconnected
}

// ClosedTable is a closed table as listed.
type ClosedTable struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Created int64  `json:"created"` // unix ms
	Closed  int64  `json:"closed"`  // unix ms
	Reason  string `json:"reason"`  // like "ended", "idle" or "empty"
	Hands   int    `json:"hands"`   // the last hand's number
}

// AdminClosed is what's kept of a closed table: how it ended.
type AdminClosed struct {
	Settings table.Settings    `json:"settings"`
	Ledger   *table.LedgerView `json:"ledger"` // nil when it was saved by another version
	Logs     []table.Log       `json:"logs"`   // the last adminLogs
}

// AuditEntry is something a connection did or tried, for the operator: who,
// from where, and whether it was refused. Players never see these.
type AuditEntry struct {
	ID        int64           `json:"id"`        // assigned when written, in order
	Time      int64           `json:"time"`      // unix ms
	Table     string          `json:"table"`     // table id, empty before joining one
	TableName string          `json:"tableName"` // empty when the message named none
	Person    string          `json:"person"`    // person id, empty before joining
	Name      string          `json:"name"`
	IP        string          `json:"ip"`
	Conn      string          `json:"conn"`                            // the connection's id
	Kind      string          `json:"kind"`                            // the command, like "take-seat", or "limit"
	Error     string          `json:"error"`                           // why it was refused, empty if it wasn't
	Data      json.RawMessage `json:"data,omitempty" tstype:"unknown"` // the command and what it logged
}
