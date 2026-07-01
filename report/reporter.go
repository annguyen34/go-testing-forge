// Package report defines how forge emits test results.
//
// The key design choice: Reporter is an interface, and the suite drives it
// (suite calls reporter, never the reverse). This is the dependency direction
// that lets you add an Allure / TeamCity / JSON reporter later without touching
// the suite. It mirrors the layered design you'd use in an API client:
// high-level orchestration depends on an abstraction, concrete outputs plug in.
package report

import "time"

// Status is the outcome of a single test.
type Status int

const (
	Passed Status = iota
	Failed
	Skipped
)

func (s Status) String() string {
	switch s {
	case Passed:
		return "PASS"
	case Failed:
		return "FAIL"
	case Skipped:
		return "SKIP"
	default:
		return "?"
	}
}

// TestResult is the outcome of one test case.
type TestResult struct {
	Name     string
	Status   Status
	Duration time.Duration
	Message  string // failure detail, empty on pass
}

// SuiteResult aggregates the tests in one suite.
type SuiteResult struct {
	Name     string
	Tests    []TestResult
	Duration time.Duration
}

// Counts returns passed, failed, skipped tallies.
func (s SuiteResult) Counts() (pass, fail, skip int) {
	for _, t := range s.Tests {
		switch t.Status {
		case Passed:
			pass++
		case Failed:
			fail++
		case Skipped:
			skip++
		}
	}
	return
}

// Reporter receives lifecycle events from a running suite. Implementations must
// be safe to call in the order: SuiteStart, then per test (TestStart, TestEnd),
// then SuiteEnd.
type Reporter interface {
	SuiteStart(name string)
	TestStart(name string)
	TestEnd(r TestResult)
	SuiteEnd(s SuiteResult)
}
