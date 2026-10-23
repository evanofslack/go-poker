package server

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/evanofslack/go-poker/table"
)

// Vars so loadConfig and tests can change them.
var (
	maxConns       = 1000    // websocket connections across all clients
	maxConnsPerIP  = 10      // websocket connections from one address
	maxTablesPerIP = 3       // open tables created from one address
	trustProxy     = false   // take the client address from X-Forwarded-For
	metricsAddr    = ":2112" // where /metrics is served, empty for off; never the public port
	retainDays     = 30      // closed tables are kept this long
	databaseURL    = ""      // postgres://… or sqlite:path; empty keeps tables in memory

	timing = table.DefaultTiming // how long tables wait for things
)

// loadConfig overrides the limits from the environment. A value that doesn't
// parse is logged and the default kept.
func loadConfig() {
	envInt("MAX_TABLES", &maxTables)
	envInt("MAX_MESSAGES_PER_SECOND", &maxMessagesPerSecond)
	envInt("MAX_CONNS", &maxConns)
	envInt("MAX_CONNS_PER_IP", &maxConnsPerIP)
	envInt("MAX_TABLES_PER_IP", &maxTablesPerIP)
	envInt("RETAIN_DAYS", &retainDays)
	envDuration("RECONNECT_GRACE", &timing.ReconnectGrace)
	envDuration("ENDED_TABLE_TTL", &timing.EndedTTL)
	envDuration("IDLE_TABLE_TIMEOUT", &timing.IdleTimeout)
	envBool("TRUST_PROXY", &trustProxy)
	databaseURL = os.Getenv("DATABASE_URL")
	if s, ok := os.LookupEnv("METRICS_ADDR"); ok {
		metricsAddr = s
	}
}

func envInt(key string, v *int) {
	if s := os.Getenv(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			*v = n
		} else {
			slog.Default().Warn("invalid config, using default", "key", key, "value", s, "default", *v)
		}
	}
}

func envDuration(key string, v *time.Duration) {
	if s := os.Getenv(key); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			*v = d
		} else {
			slog.Default().Warn("invalid config, using default", "key", key, "value", s, "default", *v)
		}
	}
}

func envBool(key string, v *bool) {
	if s := os.Getenv(key); s != "" {
		if b, err := strconv.ParseBool(s); err == nil {
			*v = b
		} else {
			slog.Default().Warn("invalid config, using default", "key", key, "value", s, "default", *v)
		}
	}
}
