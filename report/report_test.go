package report

import (
	"strings"
	"testing"
	"time"
)

func TestJUnit_Marshal(t *testing.T) {
	j := &JUnit{}
	j.SuiteEnd(SuiteResult{
		Name:     "math",
		Duration: 5 * time.Millisecond,
		Tests: []TestResult{
			{Name: "Add", Status: Passed, Duration: time.Millisecond},
			{Name: "Div", Status: Failed, Duration: 2 * time.Millisecond, Message: "expected 2, got 3"},
			{Name: "TODO", Status: Skipped},
		},
	})

	out, err := j.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	xml := string(out)

	for _, want := range []string{
		`<?xml version`,
		`<testsuite name="math"`,
		`tests="3"`,
		`failures="1"`,
		`skipped="1"`,
		`<testcase name="Add"`,
		`<failure message="expected 2, got 3"`,
		`<skipped>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("JUnit XML missing %q\n--- got ---\n%s", want, xml)
		}
	}
}

func TestSuiteResult_Counts(t *testing.T) {
	s := SuiteResult{Tests: []TestResult{
		{Status: Passed}, {Status: Passed}, {Status: Failed}, {Status: Skipped},
	}}
	pass, fail, skip := s.Counts()
	if pass != 2 || fail != 1 || skip != 1 {
		t.Fatalf("got pass=%d fail=%d skip=%d", pass, fail, skip)
	}
}
