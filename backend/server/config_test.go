package server

import (
	"testing"
	"time"
)

// reads into locals: live goroutines from other tests read the package vars
func TestEnvParsing(t *testing.T) {
	n, d, b := 1, time.Second, false

	t.Setenv("N", "7")
	t.Setenv("D", "2m")
	t.Setenv("B", "true")
	envInt("N", &n)
	envDuration("D", &d)
	envBool("B", &b)
	if n != 7 || d != 2*time.Minute || !b {
		t.Errorf("got n=%d d=%v b=%v", n, d, b)
	}

	// bad values keep what was there
	t.Setenv("N", "lots")
	t.Setenv("D", "-1s")
	t.Setenv("B", "maybe")
	envInt("N", &n)
	envDuration("D", &d)
	envBool("B", &b)
	if n != 7 || d != 2*time.Minute || !b {
		t.Errorf("bad values changed config: n=%d d=%v b=%v", n, d, b)
	}
}
