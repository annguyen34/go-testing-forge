package report

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConsole_Output(t *testing.T) {
	// Disable color so assertions match plain text regardless of environment.
	t.Setenv("NO_COLOR", "1")
	os.Unsetenv("FORCE_COLOR")

	var buf bytes.Buffer
	c := &Console{W: &buf}

	c.SuiteStart("demo")
	c.TestEnd(TestResult{Name: "passes", Status: Passed, Duration: 2 * time.Millisecond})
	c.TestEnd(TestResult{Name: "breaks", Status: Failed, Duration: time.Millisecond, Message: "expected 1, got 2"})
	c.TestEnd(TestResult{Name: "later", Status: Skipped})
	c.SuiteEnd(SuiteResult{Name: "demo", Duration: 3 * time.Millisecond, Tests: []TestResult{
		{Status: Passed}, {Status: Failed}, {Status: Skipped},
	}})

	out := buf.String()
	for _, want := range []string{
		"demo",
		"passes",
		"breaks",
		"expected 1, got 2", // failure detail printed
		"later",
		"1 passed, 1 failed, 1 skipped",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("console output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestConsole_DurationFormat(t *testing.T) {
	if got := dur(500 * time.Microsecond); !strings.HasSuffix(got, "µs") {
		t.Errorf("sub-ms duration should be in µs, got %q", got)
	}
	if got := dur(5 * time.Millisecond); !strings.HasSuffix(got, "ms") {
		t.Errorf("ms duration should be in ms, got %q", got)
	}
}
