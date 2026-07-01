package suite

import (
	"errors"
	"testing"

	"github.com/annguyen34/forge/assert"
	"github.com/annguyen34/forge/report"
)

// collectReporter records every TestResult for inspection.
type collectReporter struct{ results []report.TestResult }

func (c *collectReporter) SuiteStart(string)           {}
func (c *collectReporter) TestStart(string)            {}
func (c *collectReporter) TestEnd(r report.TestResult) { c.results = append(c.results, r) }
func (c *collectReporter) SuiteEnd(report.SuiteResult) {}

func statusOf(results []report.TestResult, name string) (report.Status, bool) {
	for _, r := range results {
		if r.Name == name {
			return r.Status, true
		}
	}
	return 0, false
}

func TestTags_FilterSelectsMatching(t *testing.T) {
	rep := &collectReporter{}
	t.Run("inner", func(t *testing.T) {
		s := New(t, "tagged", rep).OnlyTags("smoke")
		s.Run("smoke test", func(t *testing.T) {
			assert.Expect(t, 1).ToEqual(1)
		}, Tags("smoke"))
		s.Run("regression test", func(t *testing.T) {
			t.Fatal("should not run — filtered out")
		}, Tags("regression"))
		s.Run("untagged test", func(t *testing.T) {
			t.Fatal("should not run — untagged under active filter")
		})
	})

	if st, _ := statusOf(rep.results, "smoke test"); st != report.Passed {
		t.Errorf("smoke test should have passed, got %v", st)
	}
	if st, _ := statusOf(rep.results, "regression test"); st != report.Skipped {
		t.Errorf("regression test should be skipped, got %v", st)
	}
	if st, _ := statusOf(rep.results, "untagged test"); st != report.Skipped {
		t.Errorf("untagged test should be skipped under active filter, got %v", st)
	}
}

func TestTags_NoFilterRunsEverything(t *testing.T) {
	rep := &collectReporter{}
	t.Run("inner", func(t *testing.T) {
		s := New(t, "nofilter", rep)
		s.Run("a", func(t *testing.T) {}, Tags("x"))
		s.Run("b", func(t *testing.T) {}) // untagged
	})
	if st, _ := statusOf(rep.results, "a"); st != report.Passed {
		t.Errorf("a should run, got %v", st)
	}
	if st, _ := statusOf(rep.results, "b"); st != report.Passed {
		t.Errorf("b should run, got %v", st)
	}
}

func TestSkip_RecordsReason(t *testing.T) {
	rep := &collectReporter{}
	t.Run("inner", func(t *testing.T) {
		s := New(t, "skips", rep)
		s.Run("wip", func(t *testing.T) {
			t.Fatal("skipped test body must not run")
		}, Skip("not implemented yet"))
	})
	st, found := statusOf(rep.results, "wip")
	if !found || st != report.Skipped {
		t.Fatalf("wip should be skipped, got %v found=%v", st, found)
	}
}

// flakyFunc fails the first (failUntil-1) times, then passes.
func flakyFunc(failUntil int) func(t assert.TB) {
	calls := 0
	return func(t assert.TB) {
		calls++
		var err error
		if calls < failUntil {
			err = errors.New("transient")
		}
		assert.Expect(t, err).ToBeNil()
	}
}

func TestRetry_PassesEventually(t *testing.T) {
	rep := &collectReporter{}
	t.Run("inner", func(t *testing.T) {
		s := New(t, "flaky", rep)
		// Fails twice then passes on the 3rd attempt; allow 3 attempts.
		s.RunFlaky("eventually green", 3, flakyFunc(3))
	})
	st, found := statusOf(rep.results, "eventually green")
	if !found || st != report.Passed {
		t.Fatalf("flaky test should pass within 3 attempts, got %v found=%v", st, found)
	}
}

func TestRetry_FailsWhenAttemptsExhausted(t *testing.T) {
	// Test the pure verdict logic directly so the expected failure does not
	// propagate into go test and fail this test. evalFlaky never touches a real
	// *testing.T.
	s := &Suite{}
	passed, used, msg := s.evalFlaky(2, flakyFunc(5)) // needs 5, only 2 attempts
	if passed {
		t.Fatal("expected flaky test to fail when attempts are exhausted")
	}
	if used != 2 {
		t.Errorf("expected 2 attempts used, got %d", used)
	}
	if msg == "" {
		t.Error("expected a failure message from the last attempt")
	}
}

func TestRetry_EvalPassesAndReportsAttempt(t *testing.T) {
	s := &Suite{}
	passed, used, _ := s.evalFlaky(5, flakyFunc(3)) // passes on 3rd
	if !passed || used != 3 {
		t.Fatalf("expected pass on attempt 3, got passed=%v used=%d", passed, used)
	}
}
