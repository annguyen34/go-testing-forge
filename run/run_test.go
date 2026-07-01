package run

import (
	"strings"
	"testing"

	"github.com/annguyen34/forge/report"
)

// A canned test2json stream: one package, three tests (pass, fail, skip).
const stream = `
{"Action":"run","Package":"example/math","Test":"TestAdd"}
{"Action":"output","Package":"example/math","Test":"TestAdd","Output":"=== RUN   TestAdd\n"}
{"Action":"pass","Package":"example/math","Test":"TestAdd","Elapsed":0.01}
{"Action":"run","Package":"example/math","Test":"TestDiv"}
{"Action":"output","Package":"example/math","Test":"TestDiv","Output":"=== RUN   TestDiv\n"}
{"Action":"output","Package":"example/math","Test":"TestDiv","Output":"    div_test.go:9: expected 2, got 3\n"}
{"Action":"output","Package":"example/math","Test":"TestDiv","Output":"--- FAIL: TestDiv (0.00s)\n"}
{"Action":"fail","Package":"example/math","Test":"TestDiv","Elapsed":0.0}
{"Action":"run","Package":"example/math","Test":"TestTODO"}
{"Action":"skip","Package":"example/math","Test":"TestTODO","Elapsed":0.0}
{"Action":"fail","Package":"example/math","Elapsed":0.02}
`

func TestParse_AggregatesByPackage(t *testing.T) {
	suites, err := Parse(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}
	s := suites[0]
	if s.Name != "example/math" {
		t.Errorf("suite name = %q", s.Name)
	}
	pass, fail, skip := s.Counts()
	if pass != 1 || fail != 1 || skip != 1 {
		t.Fatalf("counts pass=%d fail=%d skip=%d", pass, fail, skip)
	}

	byName := map[string]report.TestResult{}
	for _, tr := range s.Tests {
		byName[tr.Name] = tr
	}
	if byName["TestDiv"].Status != report.Failed {
		t.Error("TestDiv should be Failed")
	}
	// Failure message should keep the meaningful line, drop the === RUN framing.
	msg := byName["TestDiv"].Message
	if !strings.Contains(msg, "expected 2, got 3") {
		t.Errorf("failure message missing detail: %q", msg)
	}
	if strings.Contains(msg, "=== RUN") || strings.Contains(msg, "--- FAIL") {
		t.Errorf("failure message should not contain go test framing: %q", msg)
	}
}

func TestParse_SkipsNonJSON(t *testing.T) {
	noisy := "# example/math\nsome build noise\n" + stream
	suites, err := Parse(strings.NewReader(noisy))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite despite noise, got %d", len(suites))
	}
}

func TestParse_DropsPackagesWithNoTests(t *testing.T) {
	// A package-level pass with no test events (e.g. "[no test files]").
	only := `{"Action":"pass","Package":"example/empty","Elapsed":0.0}` + "\n"
	suites, err := Parse(strings.NewReader(only))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(suites) != 0 {
		t.Fatalf("expected packages with no tests to be dropped, got %d", len(suites))
	}
}
