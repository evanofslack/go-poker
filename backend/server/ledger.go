package server

import "time"

// ledger entry kinds
const (
	entryBuyIn   = "buy-in"
	entryRebuy   = "rebuy"
	entryCashOut = "cash-out"
)

// ledgerEntry is chips coming to or leaving the table for one name.
type ledgerEntry struct {
	Time   time.Time
	Name   string
	Kind   string
	Amount uint
	By     string // who made the change: the player, or the host once hosts can edit chips
}

// ledgerRow is one name's totals. Rows are by name, so someone who leaves and
// comes back under the same name continues their row.
type ledgerRow struct {
	Name      string `json:"name"`
	BoughtIn  uint   `json:"boughtIn"`
	CashedOut uint   `json:"cashedOut"`
	Stack     uint   `json:"stack"` // chips at the table now, counting this hand's bets
	Net       int64  `json:"net"`   // stack + cashedOut - boughtIn
}

// ledgerView is the ledger as every client sees it.
type ledgerView struct {
	Rows    []ledgerRow `json:"rows"`
	Banker  string      `json:"banker"`  // name everyone settles up with
	Balance int64       `json:"balance"` // sum of nets, 0 when every chip is accounted for
}

// record adds a ledger entry. Caller holds t.mu.
func (t *table) record(name, kind string, amount uint, by string) {
	t.ledger = append(t.ledger, ledgerEntry{time.Now(), name, kind, amount, by})
}

// inLedger reports whether name has any ledger entries. Caller holds t.mu.
func (t *table) inLedger(name string) bool {
	for _, e := range t.ledger {
		if e.Name == name {
			return true
		}
	}
	return false
}

// bankerName is who everyone settles up with: the host. Caller holds t.mu.
func (t *table) bankerName() string {
	if t.host == nil {
		return ""
	}
	return t.host.username
}

// ledgerView totals the ledger, in the order names first bought in. Caller
// holds t.mu.
func (t *table) ledgerView() ledgerView {
	view := ledgerView{Rows: []ledgerRow{}, Banker: t.bankerName()}
	index := map[string]int{}
	for _, e := range t.ledger {
		i, ok := index[e.Name]
		if !ok {
			i = len(view.Rows)
			index[e.Name] = i
			view.Rows = append(view.Rows, ledgerRow{Name: e.Name})
		}
		if e.Kind == entryCashOut {
			view.Rows[i].CashedOut += e.Amount
		} else {
			view.Rows[i].BoughtIn += e.Amount
		}
	}
	for _, p := range t.game.GenerateOmniView().Players {
		if i, ok := index[p.Username]; ok {
			view.Rows[i].Stack = p.Stack + p.TotalBet
		}
	}
	for i := range view.Rows {
		r := &view.Rows[i]
		r.Net = int64(r.Stack) + int64(r.CashedOut) - int64(r.BoughtIn)
		view.Balance += r.Net
	}
	return view
}
