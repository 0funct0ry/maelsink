//go:build leakcheck

package smtp

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain applies a goroutine-leak check across every test in this
// package, per SPEC.md §2.3 point 2 (the SMTP listener is explicitly called
// out as a package where long-lived connection-handler goroutines must not
// leak). Gated behind the "leakcheck" build tag so `make test` stays fast
// and `make test-leak` opts into this stricter check.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
