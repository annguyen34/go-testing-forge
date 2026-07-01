package assert

import (
	"strings"
)

// SoftAssert collects assertion failures without aborting the test, then
// reports them all at once. This is the pattern Playwright's soft expects and
// testify's assert (vs require) give you: see every failure in one run instead
// of fixing them one at a time.
//
// Usage:
//
//	func TestThing(t *testing.T) {
//	    soft := assert.NewSoft(t)
//	    defer soft.Flush()
//
//	    soft.Expect(got.Name).ToEqual("ada")
//	    soft.Expect(got.Age).ToEqual(36)   // still runs even if the line above failed
//	}
type SoftAssert struct {
	t        TB
	failures []string
}

// NewSoft creates a soft assertion scope bound to t. Call Flush (typically via
// defer) to report collected failures.
func NewSoft(t TB) *SoftAssert {
	t.Helper()
	return &SoftAssert{t: t}
}

// Expect begins a soft assertion. Failed matchers are recorded, not fatal.
func (s *SoftAssert) Expect(actual any) *Assertion {
	s.t.Helper()
	return &Assertion{t: s.t, actual: actual, soft: s}
}

func (s *SoftAssert) record(msg string) {
	s.failures = append(s.failures, msg)
}

// Flush reports all collected failures as a single test failure. Safe to call
// when there were no failures (it does nothing). Returns true if any failures
// were recorded, so callers can branch if they want.
func (s *SoftAssert) Flush() bool {
	s.t.Helper()
	if len(s.failures) == 0 {
		return false
	}
	var b strings.Builder
	b.WriteString("soft assertions failed (")
	b.WriteString(plural(len(s.failures)))
	b.WriteString("):\n")
	for i, f := range s.failures {
		b.WriteString(indent(sprintf("%d) %s", i+1, f)))
		if i < len(s.failures)-1 {
			b.WriteString("\n")
		}
	}
	s.t.Error(b.String())
	return true
}

func plural(n int) string {
	if n == 1 {
		return "1 failure"
	}
	return sprintf("%d failures", n)
}
