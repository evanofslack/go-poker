package server

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tables, grace, proxy := maxTables, reconnectGrace, trustProxy
	t.Cleanup(func() { maxTables, reconnectGrace, trustProxy = tables, grace, proxy })

	t.Setenv("MAX_TABLES", "7")
	t.Setenv("RECONNECT_GRACE", "2m")
	t.Setenv("TRUST_PROXY", "true")
	loadConfig()
	if maxTables != 7 || reconnectGrace != 2*time.Minute || !trustProxy {
		t.Errorf("got maxTables=%d reconnectGrace=%v trustProxy=%v", maxTables, reconnectGrace, trustProxy)
	}

	// bad values keep what was there
	t.Setenv("MAX_TABLES", "lots")
	t.Setenv("RECONNECT_GRACE", "-1s")
	t.Setenv("TRUST_PROXY", "maybe")
	loadConfig()
	if maxTables != 7 || reconnectGrace != 2*time.Minute || !trustProxy {
		t.Errorf("bad values changed config: maxTables=%d reconnectGrace=%v trustProxy=%v", maxTables, reconnectGrace, trustProxy)
	}
}
