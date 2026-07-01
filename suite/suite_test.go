package suite

import (
	"testing"

	"github.com/annguyen34/forge/assert"
	"github.com/annguyen34/forge/report"
)

// recordingReporter captures the event stream so we can assert on lifecycle.
type recordingReporter struct {
	events []string
}

func (r *recordingReporter) SuiteStart(name string) { r.events = append(r.events, "suiteStart:"+name) }
func (r *recordingReporter) TestStart(name string)  { r.events = append(r.events, "testStart:"+name) }
func (r *recordingReporter) TestEnd(t report.TestResult) {
	r.events = append(r.events, "testEnd:"+t.Name+":"+t.Status.String())
}
func (r *recordingReporter) SuiteEnd(s report.SuiteResult) {
	r.events = append(r.events, "suiteEnd:"+s.Name)
}

func TestSuite_LifecycleOrder(t *testing.T) {
	var order []string
	rep := &recordingReporter{}

	// Run the suite inside a subtest so its t.Cleanup (which fires SuiteEnd)
	// completes before we inspect events.
	t.Run("inner", func(t *testing.T) {
		s := New(t, "demo", rep)
		s.BeforeAll(func() { order = append(order, "beforeAll") })
		s.AfterAll(func() { order = append(order, "afterAll") })
		s.BeforeEach(func() { order = append(order, "beforeEach") })
		s.AfterEach(func() { order = append(order, "afterEach") })

		s.Run("first", func(t *testing.T) {
			order = append(order, "test:first")
			assert.Expect(t, 1+1).ToEqual(2)
		})
		s.Run("second", func(t *testing.T) {
			order = append(order, "test:second")
			assert.Expect(t, "go").ToContain("g")
		})
	})

	want := []string{
		"beforeAll",
		"beforeEach", "test:first", "afterEach",
		"beforeEach", "test:second", "afterEach",
		"afterAll",
	}
	if len(order) != len(want) {
		t.Fatalf("lifecycle order length mismatch:\n got: %v\nwant: %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("step %d: got %q want %q\nfull: %v", i, order[i], want[i], order)
		}
	}

	// Reporter must have seen both tests pass.
	assertContains(t, rep.events, "testEnd:first:PASS")
	assertContains(t, rep.events, "testEnd:second:PASS")
	assertContains(t, rep.events, "suiteEnd:demo")
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, h := range haystack {
		if h == needle {
			return
		}
	}
	t.Errorf("expected events to contain %q, got %v", needle, haystack)
}
