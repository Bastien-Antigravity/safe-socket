// =============================================================================
// ESSENTIAL PROCESS:
// Unit tests for EnsureSafeLogger and NoOpLogger in safe-socket.
//
// DATA FLOW:
// Logger injection -> EnsureSafeLogger() -> Assertion
//
// KEY PARAMETERS:
// - STRICT_LOGGER: Environment variable toggling strict mode panic behavior.
// =============================================================================

package interfaces_test

import (
	"os"
	"testing"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// -----------------------------------------------------------------------------

func TestEnsureSafeLogger_NormalFallback(t *testing.T) {
	_ = os.Unsetenv("STRICT_LOGGER")

	l := interfaces.EnsureSafeLogger(nil)
	if l == nil {
		t.Fatal("expected non-nil logger from EnsureSafeLogger")
	}

	// Calling methods on no-op logger must not panic
	l.Debug("test debug")
	l.Info("test info")
	l.Warning("test warning")
	l.Error("test error")
	l.Critical("test critical")
	l.Close()
}

// -----------------------------------------------------------------------------

func TestEnsureSafeLogger_StrictModePanics(t *testing.T) {
	_ = os.Setenv("STRICT_LOGGER", "true")
	defer os.Unsetenv("STRICT_LOGGER")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when STRICT_LOGGER=true and logger is nil")
		}
	}()

	_ = interfaces.EnsureSafeLogger(nil)
}
