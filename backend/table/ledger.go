package table

import "time"

// ledger entry kinds
const (
	entryBuyIn   = "buy-in"
	entryRebuy   = "rebuy"
	entryCashOut = "cash-out"
	entryAdd     = "host-add"
	entryRemove  = "host-remove"
)

// LedgerEntry is chips coming to or leaving the table for one name.
type LedgerEntry struct {
	Time   time.Time `json:"time"`
	Name   string    `json:"name"`
	Kind   string    `json:"kind"`
	Amount uint      `json:"amount"`
	By     string    `json:"by"`   // who made the change: the player, or the host
	Note   string    `json:"note"` // the host's reason for a chip edit
}

// LedgerRow is one name's totals. Rows are by name, so someone who leaves and
// comes back under the same name continues their row.
type LedgerRow struct {
	Name      string `json:"name"`
	BoughtIn  uint   `json:"boughtIn"`
	CashedOut uint   `json:"cashedOut"`
	Stack     uint   `json:"stack"` // chips at the table now, counting this hand's bets
	Net       int64  `json:"net"`   // stack + cashedOut - boughtIn
}

// LedgerView is the ledger as every client sees it.
type LedgerView struct {
	Rows    []LedgerRow `json:"rows"`
	Banker  string      `json:"banker"`  // name everyone settles up with
	Balance int64       `json:"balance"` // sum of nets, 0 when every chip is accounted for
}

// record adds a ledger entry.
func (r *reducer) record(name, kind string, amount uint, by, note string) {
	r.s.Ledger = append(r.s.Ledger, LedgerEntry{r.now, name, kind, amount, by, note})
}

// inLedger reports whether name has any ledger entries.
func (s *State) inLedger(name string) bool {
	for _, e := range s.Ledger {
		if e.Name == name {
			return true
		}
	}
	return false
}

// bankerName is who everyone settles up with: the host unless the host
// picked someone.
func (s *State) bankerName() string {
	if s.Banker != "" {
		return s.Banker
	}
	return s.Host
}

// Ledger totals the ledger, in the order names first bought in.
func Ledger(s *State) LedgerView {
	view := LedgerView{Rows: []LedgerRow{}, Banker: s.bankerName()}
	index := map[string]int{}
	for _, e := range s.Ledger {
		i, ok := index[e.Name]
		if !ok {
			i = len(view.Rows)
			index[e.Name] = i
			view.Rows = append(view.Rows, LedgerRow{Name: e.Name})
		}
		if e.Kind == entryCashOut || e.Kind == entryRemove {
			view.Rows[i].CashedOut += e.Amount
		} else {
			view.Rows[i].BoughtIn += e.Amount
		}
	}
	for _, p := range s.Game.Players {
		if i, ok := index[s.Seats[p.Seat].Name]; ok {
			view.Rows[i].Stack += p.Stack + p.Committed
		}
	}
	for i := range view.Rows {
		row := &view.Rows[i]
		row.Net = int64(row.Stack) + int64(row.CashedOut) - int64(row.BoughtIn)
		view.Balance += row.Net
	}
	return view
}
