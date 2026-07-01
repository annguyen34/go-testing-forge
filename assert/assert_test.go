package assert

import (
	"errors"
	"strings"
	"testing"
)

// fakeT is a stand-in for *testing.T that records failures instead of failing.
// It lets us assert on the engine's own behavior — including the failure paths
// and messages, which is the whole point of a test for an assertion library.
type fakeT struct {
	failed bool
	msgs   []string
}

func (f *fakeT) Helper() {}
func (f *fakeT) Fatalf(format string, args ...any) {
	f.failed = true
	f.msgs = append(f.msgs, sprintf(format, args...))
	// NOTE: real t.Fatalf calls runtime.Goexit. We deliberately do NOT, so the
	// test body continues and we can inspect state. That's fine for unit-testing
	// individual matchers one per fakeT.
}
func (f *fakeT) Error(args ...any) {
	f.failed = true
	f.msgs = append(f.msgs, sprintf("%v", args...))
}
func (f *fakeT) lastMsg() string {
	if len(f.msgs) == 0 {
		return ""
	}
	return f.msgs[len(f.msgs)-1]
}

// --- happy paths: these use the REAL t, so they must genuinely pass ----------

func TestMatchers_Pass(t *testing.T) {
	Expect(t, 42).ToEqual(42)
	Expect(t, "hello").ToNotEqual("world")
	Expect(t, []int{1, 2, 3}).ToHaveLen(3)
	Expect(t, "forge testing").ToContain("test")
	Expect(t, []string{"a", "b"}).ToContain("b")
	Expect(t, map[string]int{"x": 1}).ToContain("x")
	Expect(t, "build-123").ToMatch(`build-\d+`)
	Expect(t, true).ToBeTrue()
	Expect(t, false).ToBeFalse()
	Expect(t, nil).ToBeNil()
	Expect(t, "x").ToNotBeNil()
	Expect(t, errors.New("boom")).ToError()
	Expect(t, errors.New("connection refused")).ToErrorContaining("refused")
}

// --- failure paths: these use fakeT and assert that we DID fail -------------

func TestMatchers_Fail(t *testing.T) {
	cases := []struct {
		name      string
		run       func(f *fakeT)
		wantInMsg string
	}{
		{"ToEqual mismatch", func(f *fakeT) { Expect(f, 1).ToEqual(2) }, "expected values to be equal"},
		{"ToEqual type diff shows types", func(f *fakeT) { Expect(f, 1).ToEqual(int64(1)) }, "int64"},
		{"ToNotEqual same", func(f *fakeT) { Expect(f, "a").ToNotEqual("a") }, "expected values to differ"},
		{"ToHaveLen wrong", func(f *fakeT) { Expect(f, []int{1}).ToHaveLen(3) }, "expected length 3, got 1"},
		{"ToContain missing substr", func(f *fakeT) { Expect(f, "abc").ToContain("z") }, "to contain"},
		{"ToContain missing element", func(f *fakeT) { Expect(f, []int{1, 2}).ToContain(9) }, "to contain"},
		{"ToMatch no match", func(f *fakeT) { Expect(f, "abc").ToMatch(`\d+`) }, "to match pattern"},
		{"ToBeTrue on false", func(f *fakeT) { Expect(f, false).ToBeTrue() }, "expected true"},
		{"ToBeNil on value", func(f *fakeT) { Expect(f, 5).ToBeNil() }, "expected nil"},
		{"ToError on nil", func(f *fakeT) { Expect(f, nil).ToError() }, "expected an error"},
		{"ToErrorContaining wrong", func(f *fakeT) { Expect(f, errors.New("a")).ToErrorContaining("b") }, "to contain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeT{}
			tc.run(f)
			if !f.failed {
				t.Fatalf("expected a failure but matcher passed")
			}
			if !strings.Contains(f.lastMsg(), tc.wantInMsg) {
				t.Fatalf("failure message %q did not contain %q", f.lastMsg(), tc.wantInMsg)
			}
		})
	}
}

// --- soft assertions collect and flush -------------------------------------

func TestSoftAssert_CollectsAll(t *testing.T) {
	f := &fakeT{}
	soft := NewSoft(f)
	soft.Expect(1).ToEqual(2) // fail 1
	soft.Expect(3).ToEqual(4) // fail 2 — must still run despite fail 1
	if f.failed {
		t.Fatal("soft assertions should not fail until Flush")
	}
	flushed := soft.Flush()
	if !flushed || !f.failed {
		t.Fatal("Flush should report collected failures")
	}
	if !strings.Contains(f.lastMsg(), "2 failures") {
		t.Fatalf("expected message to mention 2 failures, got: %q", f.lastMsg())
	}
}

func TestSoftAssert_NoFailuresIsSilent(t *testing.T) {
	f := &fakeT{}
	soft := NewSoft(f)
	soft.Expect(1).ToEqual(1)
	if soft.Flush() {
		t.Fatal("Flush should return false when nothing failed")
	}
	if f.failed {
		t.Fatal("no failure should have been reported")
	}
}
