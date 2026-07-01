// Package suite organizes tests, manages setup/teardown lifecycle, and drives
// a Reporter — all while staying on top of testing.T so `go test` remains the
// runner.
//
// The lifecycle mirrors what you already know from pytest/Playwright:
//
//	BeforeAll   → once before any test in the suite
//	  BeforeEach → before each test
//	    <test>
//	  AfterEach  → after each test (runs even on failure, via t.Cleanup)
//	AfterAll    → once after all tests
//
// Tests can carry tags for selective execution (see options.go) and flaky tests
// can be retried (see retry.go).
package suite

import (
	"testing"
	"time"

	"github.com/annguyen34/forge/report"
)

// nowFunc is a seam so tests could fake time if needed; defaults to time.Now.
var nowFunc = time.Now

// Suite holds shared lifecycle hooks and a reporter. Construct with New, then
// register hooks, then call Run for each test.
type Suite struct {
	name     string
	t        *testing.T
	reporter report.Reporter

	beforeAll  func()
	afterAll   func()
	beforeEach func()
	afterEach  func()

	filter    []string // active tag filter (via OnlyTags)
	filterSet bool

	results   []report.TestResult
	startedAt time.Time
	ranAll    bool
}

// New creates a suite bound to t. If reporter is nil, a console reporter is used.
func New(t *testing.T, name string, reporter report.Reporter) *Suite {
	t.Helper()
	if reporter == nil {
		reporter = &report.Console{}
	}
	s := &Suite{name: name, t: t, reporter: reporter, startedAt: nowFunc()}
	s.reporter.SuiteStart(name)
	// Guarantee SuiteEnd + AfterAll fire once, even if a test calls FailNow.
	t.Cleanup(s.finish)
	return s
}

// BeforeAll registers a function run once before the first Run.
func (s *Suite) BeforeAll(fn func()) *Suite { s.beforeAll = fn; return s }

// AfterAll registers a function run once after the suite finishes.
func (s *Suite) AfterAll(fn func()) *Suite { s.afterAll = fn; return s }

// BeforeEach registers a function run before every Run.
func (s *Suite) BeforeEach(fn func()) *Suite { s.beforeEach = fn; return s }

// AfterEach registers a function run after every Run (even on failure).
func (s *Suite) AfterEach(fn func()) *Suite { s.afterEach = fn; return s }

// Run executes a single test case as a subtest. The fn receives the subtest's
// *testing.T, so assertions inside report against the right node and parallelism
// / -run filtering keep working. Pass options for tags or skip:
//
//	s.Run("login", fn, suite.Tags("smoke"))
//	s.Run("wip",   fn, suite.Skip("not implemented"))
func (s *Suite) Run(name string, fn func(t *testing.T), opts ...Opt) {
	s.t.Helper()
	o := buildOpts(opts)

	if skip, reason := s.resolveSkip(o); skip {
		s.runSkipped(name, reason)
		return
	}
	s.ensureBeforeAll()

	s.reporter.TestStart(name)
	start := nowFunc()

	ok := s.t.Run(name, func(t *testing.T) {
		if s.afterEach != nil {
			t.Cleanup(s.afterEach) // runs after fn even if it fails/panics
		}
		if s.beforeEach != nil {
			s.beforeEach()
		}
		fn(t)
	})

	res := makeResult(name, start, ok)
	s.record(res)
	s.reporter.TestEnd(res)
}

// --- shared helpers (used by Run and RunFlaky) ------------------------------

func (s *Suite) ensureBeforeAll() {
	if s.beforeAll != nil && !s.ranAll {
		s.beforeAll()
		s.ranAll = true
	}
}

// resolveSkip combines an explicit Skip option with the active tag filter.
func (s *Suite) resolveSkip(o testOpts) (bool, string) {
	if o.hasSkip {
		return true, o.skipReason
	}
	return s.shouldSkipForTags(o.tags)
}

// runSkipped marks the subtest skipped (so go test counts it correctly) and
// records a skipped result for reporters.
func (s *Suite) runSkipped(name, reason string) {
	s.reporter.TestStart(name)
	start := nowFunc()
	s.t.Run(name, func(t *testing.T) { t.Skip(reason) })
	res := report.TestResult{
		Name:     name,
		Status:   report.Skipped,
		Duration: nowFunc().Sub(start),
		Message:  reason,
	}
	s.record(res)
	s.reporter.TestEnd(res)
}

func (s *Suite) record(r report.TestResult) {
	s.results = append(s.results, r)
}

func makeResult(name string, start time.Time, ok bool) report.TestResult {
	status := report.Passed
	if !ok {
		status = report.Failed
	}
	return report.TestResult{Name: name, Status: status, Duration: nowFunc().Sub(start)}
}

func (s *Suite) finish() {
	if s.afterAll != nil {
		s.afterAll()
	}
	s.reporter.SuiteEnd(report.SuiteResult{
		Name:     s.name,
		Tests:    s.results,
		Duration: nowFunc().Sub(s.startedAt),
	})
}
