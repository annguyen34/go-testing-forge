// Package run is the engine behind the standalone `forge` runner. Rather than
// reimplementing Go's test discovery and compilation (fragile, and a poor use of
// time), it orchestrates `go test -json` and parses the test2json event stream
// into forge's own result model — the same approach gotestsum uses. The payoff:
// forge can emit a unified console summary and JUnit XML for ANY `go test` run,
// across all packages, without the test files importing forge at all.
package run

import (
	"bufio"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/annguyen34/forge/report"
)

// event mirrors the JSON objects emitted by `go test -json` (the test2json
// format). Only the fields we use are listed.
type event struct {
	Action  string  // run, pass, fail, skip, output, start, pause, cont
	Package string  // import path
	Test    string  // test name; empty for package-level events
	Elapsed float64 // seconds, present on pass/fail/skip
	Output  string  // present on output
}

type testAcc struct {
	name   string
	status report.Status
	dur    time.Duration
	output []string
	done   bool
}

// Parse reads a test2json stream and aggregates it into one SuiteResult per
// package. It is pure (no exec), so it can be tested with a canned stream.
func Parse(r io.Reader) ([]report.SuiteResult, error) {
	// package -> test name -> accumulator
	pkgs := map[string]map[string]*testAcc{}
	pkgDur := map[string]time.Duration{}
	order := []string{} // package discovery order

	touchPkg := func(p string) map[string]*testAcc {
		m, ok := pkgs[p]
		if !ok {
			m = map[string]*testAcc{}
			pkgs[p] = m
			order = append(order, p)
		}
		return m
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // allow long output lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue // non-JSON noise (e.g. build output)
		}
		var ev event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip malformed lines rather than abort the whole run
		}
		m := touchPkg(ev.Package)

		if ev.Test == "" {
			// Package-level event: capture total duration on terminal actions.
			if ev.Action == "pass" || ev.Action == "fail" || ev.Action == "skip" {
				pkgDur[ev.Package] = time.Duration(ev.Elapsed * float64(time.Second))
			}
			continue
		}

		acc, ok := m[ev.Test]
		if !ok {
			acc = &testAcc{name: ev.Test, status: report.Passed}
			m[ev.Test] = acc
		}
		switch ev.Action {
		case "output":
			acc.output = append(acc.output, ev.Output)
		case "pass":
			acc.status, acc.dur, acc.done = report.Passed, secs(ev.Elapsed), true
		case "fail":
			acc.status, acc.dur, acc.done = report.Failed, secs(ev.Elapsed), true
		case "skip":
			acc.status, acc.dur, acc.done = report.Skipped, secs(ev.Elapsed), true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	var suites []report.SuiteResult
	for _, p := range order {
		m := pkgs[p]
		names := make([]string, 0, len(m))
		for n := range m {
			names = append(names, n)
		}
		sort.Strings(names)

		s := report.SuiteResult{Name: p, Duration: pkgDur[p]}
		for _, n := range names {
			acc := m[n]
			tr := report.TestResult{Name: acc.name, Status: acc.status, Duration: acc.dur}
			if acc.status == report.Failed {
				tr.Message = cleanOutput(acc.output)
			}
			s.Tests = append(s.Tests, tr)
		}
		// Skip packages with no test events (e.g. "[no test files]").
		if len(s.Tests) > 0 {
			suites = append(suites, s)
		}
	}
	return suites, nil
}

func secs(f float64) time.Duration { return time.Duration(f * float64(time.Second)) }

// cleanOutput joins captured output lines, trimming the go test framing noise
// ("=== RUN", "--- FAIL") to leave the meaningful failure message.
func cleanOutput(lines []string) string {
	var keep []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" ||
			strings.HasPrefix(t, "=== RUN") ||
			strings.HasPrefix(t, "=== PAUSE") ||
			strings.HasPrefix(t, "=== CONT") ||
			strings.HasPrefix(t, "--- FAIL") ||
			strings.HasPrefix(t, "--- PASS") ||
			t == "FAIL" {
			continue
		}
		keep = append(keep, t)
	}
	return strings.Join(keep, "\n")
}
