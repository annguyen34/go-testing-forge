// Package assert is the assertion engine at the heart of forge.
//
// Design decisions (the part worth understanding, and worth blogging about):
//
//  1. We build ON TOP of testing.TB instead of replacing it. Every Assertion
//     holds a testing.TB and reports failures through t.Error / t.Fatal. This
//     means `go test` stays the runner and we inherit subtests, -run, -v,
//     parallelism, and CI integration for free.
//
//  2. Expect() returns a *Assertion so matchers can chain:
//     Expect(t, x).ToEqual(1).ToBeNumeric(). Each matcher returns the same
//     *Assertion.
//
//  3. Hard vs soft: a hard assertion calls t.Fatal-style failure (stops the
//     test). A soft assertion records the failure and lets the test continue,
//     flushing all collected failures at the end. See soft.go.
package assert

// TB is the narrow slice of *testing.T that the engine actually needs.
//
// Why not just use testing.TB? Because testing.TB has an unexported method
// specifically to stop anyone outside the testing package from implementing it
// — which would make this engine untestable (we couldn't build a fake to verify
// failure paths). Defining our own minimal interface is the idiomatic fix and
// the reason testify accepts its own TestingT. *testing.T satisfies this for
// free, so callers pass `t` exactly as before.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
	Error(args ...any)
}

// Assertion is a fluent wrapper around a single actual value under test.
type Assertion struct {
	t      TB
	actual any
	soft   *SoftAssert // non-nil only for soft assertions
}

// Expect begins a hard assertion on actual. A failed matcher fails the test
// immediately (like require in testify).
//
//	Expect(t, got).ToEqual(want)
func Expect(t TB, actual any) *Assertion {
	t.Helper()
	return &Assertion{t: t, actual: actual}
}

// fail reports a failure. Hard assertions abort the test (FailNow); soft
// assertions record the message and continue.
func (a *Assertion) fail(format string, args ...any) {
	a.t.Helper()
	if a.soft != nil {
		a.soft.record(sprintf(format, args...))
		return
	}
	a.t.Fatalf(format, args...)
}
