package suite

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/annguyen34/forge/assert"
)

// captureTB is a TB that records the first failure instead of touching a real
// *testing.T. It is the key to retry: we run a flaky test body against this,
// inspect the outcome, and decide whether to retry — all without committing a
// failure to go test until we've made the final call.
//
// Subtlety worth understanding (and blogging about): a real Fatalf must stop the
// test, which testing implements via runtime.Goexit. We mirror that here, which
// is exactly why each attempt runs in its own goroutine (below) — Goexit unwinds
// and terminates only that goroutine, leaving us free to retry.
type captureTB struct {
	failed bool
	msg    string
}

func (c *captureTB) Helper() {}

func (c *captureTB) Fatalf(format string, args ...any) {
	if !c.failed {
		c.failed = true
		c.msg = fmt.Sprintf(format, args...)
	}
	runtime.Goexit()
}

func (c *captureTB) Error(args ...any) {
	if !c.failed {
		c.failed = true
		c.msg = fmt.Sprint(args...)
	}
}

// Compile-time check that captureTB satisfies the assertion engine's interface.
var _ assert.TB = (*captureTB)(nil)

// runAttempt executes fn against a fresh captureTB in a goroutine, returning
// whether it passed and the failure message if not. The goroutine is required
// because fn may call Fatalf -> Goexit.
func runAttempt(fn func(t assert.TB)) (passed bool, msg string) {
	c := &captureTB{}
	done := make(chan struct{})
	go func() {
		// If fn panics (not a test failure but a real bug), record it rather
		// than crashing the whole run; treat as a failed attempt.
		defer func() {
			if r := recover(); r != nil {
				c.failed = true
				c.msg = fmt.Sprintf("panic: %v", r)
			}
			close(done)
		}()
		fn(c)
	}()
	<-done
	return !c.failed, c.msg
}

// evalFlaky runs the attempt loop without touching *testing.T, returning the
// verdict. Splitting this out from RunFlaky makes the exhausted-failure path
// unit-testable: we can assert on (passed, used, msg) without a real subtest
// failure propagating up and failing the test that is testing us.
func (s *Suite) evalFlaky(attempts int, fn func(t assert.TB)) (passed bool, used int, lastMsg string) {
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		used = i + 1
		if s.beforeEach != nil {
			s.beforeEach()
		}
		ok, msg := runAttempt(fn)
		if s.afterEach != nil {
			s.afterEach()
		}
		if ok {
			return true, used, ""
		}
		lastMsg = msg
	}
	return false, used, lastMsg
}

// RunFlaky runs a test that may be intermittently failing, retrying up to
// `attempts` times; it passes if any attempt passes. Note the body signature is
// func(t assert.TB), not func(t *testing.T): retry requires a TB we can swap, so
// the body asserts via the same assert.Expect API but against a controllable
// target. Use plain Run for ordinary tests.
//
//	s.RunFlaky("network ping", 3, func(t assert.TB) {
//	    assert.Expect(t, ping()).ToBeNil()
//	})
func (s *Suite) RunFlaky(name string, attempts int, fn func(t assert.TB), opts ...Opt) {
	s.t.Helper()
	o := buildOpts(opts)

	if skip, reason := s.resolveSkip(o); skip {
		s.runSkipped(name, reason)
		return
	}
	s.ensureBeforeAll()

	s.reporter.TestStart(name)
	start := nowFunc()

	passed, used, lastMsg := s.evalFlaky(attempts, fn)

	// Commit a single verdict to go test via a subtest, so -v output and exit
	// code stay correct.
	s.t.Run(name, func(t *testing.T) {
		if !passed {
			t.Errorf("flaky test failed after %d attempt(s): %s", used, lastMsg)
		}
	})

	res := makeResult(name, start, passed)
	if !passed {
		res.Message = fmt.Sprintf("failed after %d attempt(s): %s", used, lastMsg)
	} else if used > 1 {
		res.Message = fmt.Sprintf("passed on attempt %d/%d", used, attempts)
	}
	s.record(res)
	s.reporter.TestEnd(res)
}
