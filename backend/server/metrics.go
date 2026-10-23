package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Labels are fixed sets, never a table, address or player.
var (
	limitLabels       = []string{"conns", "conns_per_ip", "tables", "tables_per_ip", "messages"}
	closeReasonLabels = []string{"empty", "ended", "idle", "shutdown"}
	graceLabels       = []string{"reconnected", "expired"}
)

var (
	handsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poker_hands_total", Help: "Hands dealt.",
	})
	limitHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "poker_limit_hits_total", Help: "Connections, tables or messages refused by a limit.",
	}, []string{"limit"})
	slowClientDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poker_slow_client_drops_total", Help: "Clients dropped for not keeping up with messages.",
	})
	tablesClosed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "poker_tables_closed_total", Help: "Tables closed, by reason.",
	}, []string{"reason"})
	seatGrace = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "poker_seat_grace_total", Help: "Disconnected players who reconnected or lost their seat.",
	}, []string{"outcome"})
	panicsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poker_panics_total", Help: "Panics recovered while handling a message.",
	})
)

// start every series at 0 so rate() works before the first event
func init() {
	for _, l := range limitLabels {
		limitHits.WithLabelValues(l)
	}
	for _, r := range closeReasonLabels {
		tablesClosed.WithLabelValues(r)
	}
	for _, o := range graceLabels {
		seatGrace.WithLabelValues(o)
	}
}

var (
	connectionsDesc = prometheus.NewDesc("poker_connections", "Open websocket connections.", nil, nil)
	tablesDesc      = prometheus.NewDesc("poker_tables", "Open tables, by whether their session ended.", []string{"state"}, nil)
	playersDesc     = prometheus.NewDesc("poker_players", "Seated players.", []string{"kind"}, nil)
	limitDesc       = prometheus.NewDesc("poker_limit", "Configured limits.", []string{"limit"}, nil)
)

// hubCollector reads the gauges from the hub at scrape time.
type hubCollector struct{ h *Hub }

func (c hubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- connectionsDesc
	ch <- tablesDesc
	ch <- playersDesc
	ch <- limitDesc
}

func (c hubCollector) Collect(ch chan<- prometheus.Metric) {
	c.h.connMu.Lock()
	conns := c.h.total
	c.h.connMu.Unlock()

	var active, ended, humans, bots float64
	for _, a := range c.h.allTables() {
		if a.ended.Load() {
			ended++
		} else {
			active++
		}
		s := a.summary.Load()
		humans += float64(s.Players - s.Bots)
		bots += float64(s.Bots)
	}

	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
	gauge(connectionsDesc, float64(conns))
	gauge(tablesDesc, active, "active")
	gauge(tablesDesc, ended, "ended")
	gauge(playersDesc, humans, "human")
	gauge(playersDesc, bots, "bot")
	gauge(limitDesc, float64(maxConns), "conns")
	gauge(limitDesc, float64(maxConnsPerIP), "conns_per_ip")
	gauge(limitDesc, float64(maxTables), "tables")
	gauge(limitDesc, float64(maxTablesPerIP), "tables_per_ip")
	gauge(limitDesc, float64(maxMessagesPerSecond), "messages")
}

// metricsHandler serves the game metrics plus the go and process ones.
func metricsHandler(h *Hub) http.Handler {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		hubCollector{h},
		handsTotal, limitHits, slowClientDrops, tablesClosed, seatGrace, panicsTotal,
	)
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
