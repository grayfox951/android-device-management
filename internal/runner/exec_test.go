package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCapturesBothStreams(t *testing.T) {
	// stdout and stderr must both reach the caller so adb diagnostics are
	// never silently dropped.
	res := Run(context.Background(), "sh", "-c", "echo out; echo err 1>&2")
	if !res.OK() {
		t.Errorf("command failed: %v (code %d)", res.Err, res.Code)
	}
	out := res.Out
	if !strings.Contains(out, "out") || !strings.Contains(out, "err") {
		t.Errorf("output %q is missing one of the streams", out)
	}
}

func TestRunReportsExitCode(t *testing.T) {
	res := Run(context.Background(), "sh", "-c", "exit 3")
	if res.Code != 3 {
		t.Errorf("exit code = %d, want 3", res.Code)
	}
	if res.OK() {
		t.Error("OK() should be false for a non-zero exit")
	}
}

func TestRunMissingBinary(t *testing.T) {
	res := Run(context.Background(), "definitely-not-a-real-binary-xyz")
	if res.Err == nil {
		t.Error("expected an error for a missing binary")
	}
	if res.Code != -1 {
		t.Errorf("code = %d, want -1 when the process never started", res.Code)
	}
}

func TestRunHonoursContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	res := Run(ctx, "sleep", "5")
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("command ran for %s, context deadline was ignored", elapsed)
	}
	if res.Code == 0 {
		t.Error("cancelled command reported success")
	}
}

func TestRunQuietAppliesItsOwnDeadline(t *testing.T) {
	// No deadline on the context: RunQuiet must not hang forever.
	start := time.Now()
	RunQuiet(context.Background(), "sh", "-c", "echo hi")
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("RunQuiet took %s", elapsed)
	}
}

func TestResultEmpty(t *testing.T) {
	if !(Result{Out: "   \n\t"}).Empty() {
		t.Error("whitespace-only output should count as empty")
	}
	if (Result{Out: "x"}).Empty() {
		t.Error("non-whitespace output should not count as empty")
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote([]string{"adb", "-s", "ABC", "shell", "id"}); got != "adb -s ABC shell id" {
		t.Errorf("quote = %q", got)
	}
	if got := shellQuote([]string{"echo", "a b"}); !strings.Contains(got, `"a b"`) {
		t.Errorf("an argument with a space was not quoted: %q", got)
	}
}
