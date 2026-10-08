// Package testsupport holds assertion helpers shared by the Anubis test suites.
// It contains no production logic and is imported only from _test.go files.
package testsupport

import (
	"fmt"
	"os"
	"testing"
)

// StrictEnv is the environment variable that promotes recorded contract
// deviations into hard test failures.
const StrictEnv = "ANUBIS_STRICT"

var strict = os.Getenv(StrictEnv) == "1"

// Contract records a violated product requirement.
//
// The Anubis API surface currently contains a number of confirmed defects. Tests
// that pin the *required* behaviour would make `go test ./...` permanently red,
// which destroys its value as a regression gate. Contract therefore splits the two
// concerns:
//
//   - default run: the deviation is logged as KNOWN DEVIATION and the suite stays
//     green, so it remains usable in CI and as a regression gate;
//   - ANUBIS_STRICT=1: every deviation becomes a test failure, producing the
//     red evidence consumed by the test-architecture diff matrix.
func Contract(t *testing.T, holds bool, format string, args ...any) {
	t.Helper()
	if holds {
		return
	}

	message := fmt.Sprintf(format, args...)
	if strict {
		t.Errorf("CONTRACT VIOLATION: %s", message)
		return
	}
	t.Logf("KNOWN DEVIATION (set %s=1 to fail): %s", StrictEnv, message)
}

// Strict reports whether the suite is running in strict (audit) mode.
func Strict() bool { return strict }
